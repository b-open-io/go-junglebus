module github.com/b-open-io/go-junglebus

go 1.23

require (
	github.com/centrifugal/centrifuge-go v0.10.4
	github.com/stretchr/testify v1.10.0
	google.golang.org/protobuf v1.36.4
)

require (
	github.com/centrifugal/protocol v0.14.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/segmentio/encoding v0.4.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Force all dependencies to use a newer version of x/net to avoid CVE-2024-45338
// This is needed because github.com/planetscale/vtprotobuf pulls in an older vulnerable version
replace golang.org/x/net => golang.org/x/net v0.34.0
