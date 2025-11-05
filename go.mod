module WBTechL2

go 1.24.1

require github.com/diekinari/ntpclient v0.0.0

require (
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
)

replace github.com/diekinari/ntpclient => ./ntpClient
