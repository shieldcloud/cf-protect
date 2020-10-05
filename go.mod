module github.com/shieldcloud/cf-protect

go 1.14

replace github.com/shieldproject/shield => github.com/shieldcloud/shield v0.0.0-20200924141532-2ed349e37578

require (
	code.cloudfoundry.org/cli v7.1.0+incompatible // indirect
	github.com/cloudfoundry/cli v7.1.0+incompatible
	github.com/jhunt/go-ansi v0.0.0-20181127194324-5fd839f108b6
	github.com/jhunt/vcaptive v0.0.0-20190330221511-a1b4af624bb5
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/shieldproject/shield v0.0.0-00010101000000-000000000000
)
