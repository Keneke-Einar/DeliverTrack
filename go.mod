module github.com/keneke/delivertrack

go 1.25

replace (
	github.com/keneke/delivertrack/internal/auth => ./internal/auth
	github.com/keneke/delivertrack/pkg/mongodb => ./pkg/mongodb
	github.com/keneke/delivertrack/pkg/postgres => ./pkg/postgres
)
