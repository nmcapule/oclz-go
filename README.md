# oclz-go

## Building

```sh
export REPLICA_URL=<your replica url>
export LITESTREAM_ACCESS_KEY_ID=<your s3 access key id>
export LITESTREAM_SECRET_ACCESS_KEY=<your s3 secret access key>

docker build -t latest .

docker run \
  -p 8080:8080 \
  -e REPLICA_URL \
  -e LITESTREAM_ACCESS_KEY_ID \
  -e LITESTREAM_SECRET_ACCESS_KEY \
  latest
```

## Deploying to Fly.io

```sh
flyctl secrets set REPLICA_URL=<your replica url>
flyctl secrets set LITESTREAM_ACCESS_KEY_ID=<your s3 access key id>
flyctl secrets set LITESTREAM_SECRET_ACCESS_KEY=<your s3 secret access key>

flyctl deploy
```
