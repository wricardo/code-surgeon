# code-surgeon

Code Surgeon is a library for generating code, parsing code. It offers functionality as a library, as a webservice and as a command line tool.


## Dependencies

- https://comby.dev/docs/get-started

## Using the GPT Server

- Install the code-surgeon binary to your path. `make install`
- Change directory to the root of the project you want to generate code for.
- Run `code-surgeon server` to start the server.
- On another tab, run `code-surgeon openapi-json | pbcopy` to copy the OpenAPI JSON to your clipboard. Use that to configure your chatGpt "Actions" in the OpenAI console. https://openai.com/index/introducing-gpts/
- Run `code-surgeon instructions` to the Instructions test that goes in the customGPT (same page as the step above)

