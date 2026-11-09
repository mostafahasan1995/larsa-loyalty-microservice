docker rm loyalty
docker build -t loyalty .
docker run --name loyalty  --network=host loyalty