package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	authAdapters "github.com/Keneke-Einar/delivertrack/pkg/auth/adapters"
	authApp "github.com/Keneke-Einar/delivertrack/pkg/auth/app"
	authPorts "github.com/Keneke-Einar/delivertrack/pkg/auth/ports"

	"github.com/Keneke-Einar/delivertrack/pkg/config"
	"github.com/Keneke-Einar/delivertrack/pkg/logger"
	"github.com/Keneke-Einar/delivertrack/pkg/messaging"
	"github.com/Keneke-Einar/delivertrack/pkg/postgres"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"go.uber.org/zap"
)

var version = "dev"

type Gateway struct {
	authService authPorts.AuthService
	logger      *logger.Logger
}

func main() {
	cfg, err := config.Load("gateway")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	port := cfg.Service.Port
	databaseURL := cfg.Database.URL
	jwtSecret := cfg.Auth.JWTSecret

	// Initialize logger
	lg, err := logger.NewLogger(cfg.Logging, "gateway")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Initialize database for auth
	db, err := postgres.New(databaseURL)
	if err != nil {
		lg.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	lg.Info("Database connection established")

	// Initialize auth service
	userRepo := authAdapters.NewPostgresUserRepository(db.DB)
	tokenService := authAdapters.NewJWTTokenService(jwtSecret, cfg.Auth.JWTExpiration)
	authService := authApp.NewAuthService(userRepo, tokenService)

	gateway := &Gateway{
		authService: authService,
		logger:      lg,
	}

	// Setup rate limiter
	limiter := tollbooth.NewLimiter(10, nil) // 10 requests per second
	limiter.SetIPLookups([]string{"X-Real-IP", "X-Forwarded-For", "RemoteAddr"})
	limiter.SetMethods([]string{"GET", "POST", "PUT", "DELETE"})

	// Setup router
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", gateway.healthHandler)

	// Proxy target URLs: prefer env vars (set in docker-compose), then config, then localhost fallback
	deliveryURL := os.Getenv("GATEWAY_SERVICES_DELIVERY")
	if deliveryURL == "" {
		deliveryURL = cfg.Services.Delivery
	}
	if deliveryURL == "" || !strings.HasPrefix(deliveryURL, "http") {
		deliveryURL = "http://localhost:8080"
	}
	trackingURL := os.Getenv("GATEWAY_SERVICES_TRACKING")
	if trackingURL == "" {
		trackingURL = cfg.Services.Tracking
	}
	if trackingURL == "" || !strings.HasPrefix(trackingURL, "http") {
		trackingURL = "http://localhost:8081"
	}
	notificationURL := os.Getenv("GATEWAY_SERVICES_NOTIFICATION")
	if notificationURL == "" {
		notificationURL = cfg.Services.Notification
	}
	if notificationURL == "" || !strings.HasPrefix(notificationURL, "http") {
		notificationURL = "http://localhost:8082"
	}
	analyticsURL := os.Getenv("GATEWAY_SERVICES_ANALYTICS")
	if analyticsURL == "" {
		analyticsURL = cfg.Services.Analytics
	}
	if analyticsURL == "" || !strings.HasPrefix(analyticsURL, "http") {
		analyticsURL = "http://localhost:8083"
	}

	lg.Info("Proxy targets configured",
		zap.String("delivery", deliveryURL),
		zap.String("tracking", trackingURL),
		zap.String("notification", notificationURL),
		zap.String("analytics", analyticsURL),
	)

	// API routes
	mux.Handle("/api/delivery/", gateway.authMiddleware(limiter, gateway.proxyHandler("delivery", deliveryURL)))
	mux.Handle("/api/tracking/", gateway.authMiddleware(limiter, gateway.proxyHandler("tracking", trackingURL)))
	mux.Handle("/api/notification/", gateway.authMiddleware(limiter, gateway.proxyHandler("notification", notificationURL)))
	mux.Handle("/api/analytics/", gateway.authMiddleware(limiter, gateway.proxyHandler("analytics", analyticsURL)))

	// Auth routes (public)
	authHandler := authAdapters.NewHTTPHandler(gateway.authService, cfg.Auth.JWTExpiration)
	mux.HandleFunc("/login", authHandler.Login)
	mux.HandleFunc("/register", authHandler.Register)

	// Wrap with logging and CORS
	handler := gateway.loggingMiddleware(gateway.corsMiddleware(mux))

	lg.Info("API Gateway starting", zap.String("version", version), zap.String("port", port))

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		lg.Fatal("Failed to start HTTP server", zap.Error(err))
	}
}

func (g *Gateway) proxyHandler(serviceName, targetURL string) http.HandlerFunc {
	target, _ := url.Parse(targetURL)
	proxy := httputil.NewSingleHostReverseProxy(target)

	return func(w http.ResponseWriter, r *http.Request) {
		// Rewrite path: strip /api/servicename prefix
		// e.g., /api/delivery/deliveries/ becomes /deliveries/
		prefix := "/api/" + serviceName
		r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		r.URL.Host = target.Host
		r.URL.Scheme = target.Scheme
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))

		proxy.ServeHTTP(w, r)
	}
}

func (g *Gateway) authMiddleware(lmt *limiter.Limiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Rate limiting
		httpError := tollbooth.LimitByRequest(lmt, w, r)
		if httpError != nil {
			g.logger.WithFields(
				zap.String("ip", r.RemoteAddr),
				zap.String("path", r.URL.Path),
				zap.String("error", "rate limited"),
			).Warn("Request rate limited")
			return
		}

		// Authentication
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"unauthorized","message":"Authorization header required"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"unauthorized","message":"Invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]

		claims, err := g.authService.ValidateToken(r.Context(), token)
		if err != nil {
			http.Error(w, `{"error":"unauthorized","message":"Invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "role", claims.Role)
		ctx = context.WithValue(ctx, "customer_id", claims.CustomerID)
		ctx = context.WithValue(ctx, "courier_id", claims.CourierID)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (g *Gateway) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","service":"gateway","version":"%s"}`, version)
}

func (g *Gateway) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create trace context for incoming request
		traceID := messaging.GenerateTraceID()
		spanID := messaging.GenerateSpanID()

		traceCtx := &messaging.TraceContext{
			TraceID:     traceID,
			SpanID:      spanID,
			ServiceName: "gateway",
			Operation:   r.Method + " " + r.URL.Path,
		}

		// Add trace context to request context
		ctx := messaging.ContextWithTraceContext(r.Context(), traceCtx)

		// Add trace headers to request for downstream services
		r.Header.Set("X-Trace-ID", traceID)
		r.Header.Set("X-Span-ID", spanID)

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r.WithContext(ctx))

		g.logger.WithFields(
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", wrapped.statusCode),
			zap.Duration("duration", time.Since(start)),
			zap.String("user_agent", r.Header.Get("User-Agent")),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("trace_id", traceID),
		).Info("Request processed")
	})
}

func (g *Gateway) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
