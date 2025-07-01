const jwt = require("jsonwebtoken");

const JWT_SECRET = process.env.JWT_SECRET || "your_jwt_secret";

// console.log("JWT_SECRET:", JWT_SECRET); // Log the JWT secret for debugging
// 
function authenticateJWT(req, res, next) {
  const authHeader = req.headers.authorization;

  if (!authHeader || !authHeader.startsWith("Bearer ")) {
    return res.status(401).json({ message: "Token tidak ditemukan" });
  }

  const token = authHeader.split(" ")[1];
  // console.log("Received JWT:", token); // Log the token for debugging
  try {
    const decoded = jwt.verify(token, JWT_SECRET);
    // console.log("Decoded JWT:", decoded); // Log the decoded token for debugging
    req.user = decoded; // simpan info user dari token
    next();
  } catch (err) {
    return res.status(403).json({ message: "Token tidak valid" });
  }
}

module.exports = authenticateJWT;
