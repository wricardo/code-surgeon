# code-surgeon

Code Surgeon is:
- a library for generating code, parsing golang code. 
- a command line tool to parse golang code
- an AI chatbot through grpc


## Dependencies

- https://comby.dev/docs/get-started

## Setup your env variables

- Create a `.env` file in the root of the project
- Add the following variables to the `.env` file
```
OPENAI_API_KEY=sk-....
NGROK_AUTH_TOKEN=your_ngrok_auth_token
NGROK_DOMAIN=example-domain.ngrok-free.app
NEO4j_DB_URI=neo4j://localhost
NEO4j_DB_USER=neo4j
NEO4j_DB_PASSWORD=neo4jneo4j
```

https://dashboard.ngrok.com/cloud-edge/domains

## Using chatbot


- Start infrastructure (terminal 1)
```
docker-compose up
```

- Run the chatbot server (terminal 2)
```
make run-server
```

- Run the chatbot cli client (terminal 3)
```
make new-chat
```
