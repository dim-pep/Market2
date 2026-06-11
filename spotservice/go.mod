module github.com/dim-pep/Market2/spotservice

go 1.25.4

require (
	github.com/cespare/xxhash/v2 v2.3.0
	github.com/ilyakaznacheev/cleanenv v1.5.0
	github.com/prometheus/client_golang v1.14.0
	github.com/redis/go-redis/v9 v9.20.0
	google.golang.org/protobuf v1.36.11
)

replace github.com/dim-pep/Market2/proto => ../proto

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

require (
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/dlmiddlecote/sqlstats v1.0.2
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus v1.1.0
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/lib/pq v1.12.3
	go.uber.org/zap v1.28.0
	golang.org/x/sync v0.21.0
	google.golang.org/grpc v1.81.1
	gopkg.in/yaml.v3 v3.0.1 // indirect
	olympos.io/encoding/edn v0.0.0-20201019073823-d3554ca0b0a3 // indirect
)
