# LAZYREQUESTS

A lightweight minimal API Testing Client that watches individual files or directories for changes and automatically sends HTTP requests based on .http or .rest template files.

You don't have to learn yet another Tool, if you already use restclient, you already have your http requests, because the tools uses the same systax for http requests

## Installation

```bash
go get github.com/fsnotify/fsnotify
go build -o lazyrequests ./

```


https://github.com/user-attachments/assets/6fb9089a-39a4-430f-bae7-14c9fc87da64


## Usage

```bash
./lazyrequests [options]
```

### Options

- `--watch-folder`: Folder to watch for changes
- `--watch-file`: File to watch for changes
- `--http-file`: HTTP template file to be watched and reloaded
- `--http-folder`: HTTP template folder to be watched and reloaded
- `--exclude-file`: File pattern to exclude from watching
- `--exclude-folder`: Folder to exclude from watching
- `--sleep-time`: Time to wait between each HTTP requests (milliseconds)
- `--time-out`: Timeout for each request before failing (milliseconds)
- `--verbose`: Enable verbose logging

## HTTP Template Files

Create `.http` files to define your requests. The program will parse these files and send requests based on their content. Templates support:

- HTTP method definition
- URL specification
- Custom headers
- Request body
- Expected response status (for validation)
- Comments for request identification

### Example Template

```http
###
GET http://localhost:8080/3 HTTP/1.1
Host: localhost:8080
User-Agent: curl/8.6.0
Accept: */*
###
HTTP/1.1 201 Created
Content-Type: application/json
Location: http://localhost:8080/users/3
###
```

## Example Workflow

1. Create an HTTP template file with your desired requests
2. Start the watcher pointing to your template file or directory
3. Make changes to your watched files
4. The program will automatically send the defined HTTP requests
5. View color-coded results in the terminal
