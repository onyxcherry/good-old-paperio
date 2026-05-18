# Good old paper.io game for TIwPR

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

## Run the Backend

Run the Go backend server

```bash
cd backend
```

```bash
go build -o server server.go
```

```bash
./server
```
