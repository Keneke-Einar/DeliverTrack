module github.com/keneke/delivertrack

go 1.25.5

replace (
	github.com/keneke/delivertrack/internal/delivery => ./internal/delivery
	github.com/keneke/delivertrack/pkg/auth => ./pkg/auth
	github.com/keneke/delivertrack/pkg/mongodb => ./pkg/mongodb
	github.com/keneke/delivertrack/pkg/postgres => ./pkg/postgres
)

require (
	github.com/keneke/delivertrack/internal/delivery v0.0.0-00010101000000-000000000000
	github.com/keneke/delivertrack/pkg/auth v0.0.0-00010101000000-000000000000
	github.com/keneke/delivertrack/pkg/postgres v0.0.0-00010101000000-000000000000
)

require (
	github.com/golang-jwt/jwt/v5 v5.2.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	golang.org/x/crypto v0.19.0 // indirect
)
