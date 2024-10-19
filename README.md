# MongoDB Chatbot API Wrapper

This Go code provides a wrapper around the MongoDB Knowledge API to facilitate chatbot interactions. It includes functionalities to create conversation sessions, send messages, and handle streaming responses. The code is designed to be used both as a command-line tool and as a REST API compatible with OpenAI's `/chat/completions` endpoint.

## Features

- **Conversation Management**: Create and manage conversation sessions with the MongoDB Knowledge API.
- **Message Handling**: Send messages and receive streaming responses in real-time.
- **CLI Client**: Interact with the chatbot directly from the command line.
- **REST API**: Expose a RESTful interface compatible with OpenAI's chat completion API for easy integration.

## Installation

1. **Clone the Repository**:
   ```sh
   git clone <repository-url>
   cd <repository-directory>
   ```

2. **Install Dependencies**:
   Ensure you have Go installed, then run:
   ```sh
   go mod tidy
   ```

## Usage

### Command-Line Client

To use the CLI client, run:
```sh
go run mongo.go
```
This will start an interaction with the MongoDB chatbot, sending a predefined message and printing the response.

### REST API

To start the REST API server, run:
```sh
go run mongo.go
```
The server will be available at `http://localhost:8800`.

#### Endpoints

- `GET /`: Welcome message.
- `POST /v1/chat/completions`: Send a message to the MongoDB chatbot and receive a streaming response.

#### Example Request

```sh
curl -X POST http://localhost:8800/v1/chat/completions \
-H "Content-Type: application/json" \
-d '{
  "model": "mongodb-1",
  "messages": [{"role": "user", "content": "How to use findAndModify"}],
  "stream": true
}'
```
## TODOs

- Refactor to use `net/http` instead of `gin` for the REST API.
- Implement dynamic conversation ID handling.
- Improve error handling and response validation.
- Support direct CLI
- Better error handling for when exceeding the 50 message limit

## Contributions

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## License

This project is licensed under [MIT] - see the `LICENSE` file for details.

---

**Note**: The MongoDB API doesn't require authentication, so no credentials nor configuration is required!

