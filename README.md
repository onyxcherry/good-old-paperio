# Good old paper.io game for TIwPR

## Build backend

```bash
cd backend
```

```shell
go mod tidy
```

```shell
protoc --go_out=. --go_opt=paths=source_relative pb/game.proto
```

## Run the Backend

Run the Go backend server

```bash
go build -o server .
```

```bash
./server
```

## Build the Frontend

1. Navigate to the frontend directory

   ```bash
   cd frontend
   ```

2. Install dependencies

   ```bash
   npm install
   ```

3. Build for production

   ```bash
   npm run build
   ```

   This will generate a `dist` folder.

