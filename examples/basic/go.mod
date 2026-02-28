module github.com/thecodearcher/aegis/examples/basic

go 1.24.0

require (
	github.com/gin-gonic/gin v1.11.0
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/thecodearcher/aegis v0.0.0
	github.com/thecodearcher/aegis/adapters/gorm v0.0.0
	github.com/thecodearcher/aegis/plugins/credential-password v0.0.0
	github.com/thecodearcher/aegis/plugins/oauth v0.0.0
	github.com/thecodearcher/aegis/plugins/oauth-generic v0.0.0
	github.com/thecodearcher/aegis/plugins/oauth-github v0.0.0-00010101000000-000000000000
	github.com/thecodearcher/aegis/plugins/oauth-google v0.0.0-00010101000000-000000000000
	github.com/thecodearcher/aegis/plugins/two-factor v0.0.0
	gorm.io/gorm v1.30.1
)

replace github.com/thecodearcher/aegis => ../../

replace github.com/thecodearcher/aegis/adapters/gorm => ../../adapters/gorm

replace github.com/thecodearcher/aegis/plugins/credential-password => ../../plugins/credential-password

replace github.com/thecodearcher/aegis/plugins/two-factor => ../../plugins/two-factor

replace github.com/thecodearcher/aegis/plugins/oauth => ../../plugins/oauth

replace github.com/thecodearcher/aegis/plugins/oauth-google => ../../plugins/oauth-google

replace github.com/thecodearcher/aegis/plugins/oauth-generic => ../../plugins/oauth-generic

replace github.com/thecodearcher/aegis/plugins/oauth-github => ../../plugins/oauth-github

require (
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/bytedance/sonic v1.14.0 // indirect
	github.com/bytedance/sonic/loader v0.3.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.27.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/goccy/go-yaml v1.18.0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pquerna/otp v1.4.0 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/quic-go/quic-go v0.54.0 // indirect
	github.com/simukti/sqldb-logger v0.0.0-20230108155151-646c1a075551 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.0 // indirect
	go.uber.org/mock v0.5.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/arch v0.20.0 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	golang.org/x/tools v0.38.0 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
	gorm.io/driver/postgres v1.6.0 // indirect
)
