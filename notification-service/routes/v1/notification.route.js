const {
  getNotifications,
  updateNotification,
  pushNotification,
  sendEmailNotification,
} = require("../../handlers/notification.handler");
const express = require("express");

const authenticateJWT = require("../../middlewares/authMiddleware");
const userMiddleware = require("../../middlewares/userMiddleware");

const router = express.Router();

router.get(
  "/notifications",
  [authenticateJWT, userMiddleware],
  async (req, res) => {
    await getNotifications(req, res);
  }
);

router.put(
  "/notifications/:notificationId",
  [authenticateJWT, userMiddleware],
  async (req, res) => {
    await updateNotification(req, res);
  }
);

router.post(
  "/notifications",
  [authenticateJWT, userMiddleware],
  async (req, res) => {
    await pushNotification(req, res);
  }
);

router.post(
  "/notifications/email",
  [authenticateJWT, userMiddleware],
  async (req, res) => {
    await sendEmailNotification(req, res);
  }
);

module.exports = router;
