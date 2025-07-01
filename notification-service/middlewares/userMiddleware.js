const { PrismaClient } = require("@prisma/client");
const prisma = new PrismaClient();

function userMiddleware(req, res, next) {
  const user = req.user;

  prisma.user
    .findUnique({
      where: { id: user.user_id },
    })
    .then((foundUser) => {
      if (!foundUser) {
        return prisma.user.create({
          data: {
            id: user.user_id,
            email: user.email,
            name: user.name || "User", // Default name if not provided
            // add other required fields here, e.g. email: user.email
          },
        });
      }
    })
    .then(() => {
      next();
    })
    .catch((err) => {
      res.status(500).json({ error: "Internal server error" });
    });
}


module.exports = userMiddleware;