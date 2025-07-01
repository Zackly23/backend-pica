const { setupGRPCServer } = require("./grpc/grpc");
const express = require("express");

const cors = require("cors");
const helmet = require("helmet");
const morgan = require("morgan");

const apiv1routes = require("./routes/v1/notification.route");
const fs = require("fs");
const path = require("path");

const accessLogStream = fs.createWriteStream(
  path.join(__dirname, "access.log"),
  { flags: "a" } // append
);

const app = express();
const port = 3000;


// Simpan log ke file
app.use(morgan("combined", { stream: accessLogStream }));
app.use(express.json());
app.use(cors());
app.use(helmet());
app.use("/api/v1", apiv1routes);

app.get("/health", (req, res) => {
  res.status(200).json({ status: "OK" });
}); 

app.listen(port, () => {
  console.log(`Notification service listening at http://localhost:${port}`);
  setupGRPCServer();
  console.log("gRPC server setup complete");
});
