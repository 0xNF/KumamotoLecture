# Running

## Shapeserver
* generate a TLS key certificate .key/.crt pair named `server.crt` and `server.key`
* `cd gane/shapeserver && go build`
* place the tls certificates in the same directory
* `./shapeserver`

## Shapeshifter MCP server
* `cd game/shapeshifter && go build -o shapeshifter cmd/main.go`
* `./shapeshifter --httpmode`

# MCP Connections
  * download an mcp client like 5ire or claude desktop
  * `$server=localhost`, or whatever server location you are running the binaries from
  * use the `$server/mcp` address, where $server 
  * enable the server in your client
  

# TODO
  * change http mode addr/port at runtime
  * if no certs are present, run in non-tls mode