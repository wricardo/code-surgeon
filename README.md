# code-surgeon

Code Surgeon is a library for generating code, parsing code. It offers functionality as a library, as a webservice and as a command line tool.


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
NEO4j_DB_PASSWORD=heo4j123
```

https://dashboard.ngrok.com/cloud-edge/domains

## Using the GPT Server

- Install the code-surgeon binary to your path. `make install`
- Change directory to the root of the project you want to generate code for.
- Run `code-surgeon server` to start the server.
- On another tab, run `code-surgeon openapi-json | pbcopy` to copy the OpenAPI JSON to your clipboard. Use that to configure your chatGpt "Actions" in the OpenAI console. https://openai.com/index/introducing-gpts/
- Run `code-surgeon instructions` to the Instructions test that goes in the customGPT (same page as the step above)

