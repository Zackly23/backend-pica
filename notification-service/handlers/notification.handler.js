const { PrismaClient } = require("@prisma/client");
const sendMail = require("../mails/utils/mail");
const getTemplateHTML = require("../utils/email.utils");
const prisma = new PrismaClient();

const pushNotification = async (req, res) => {
  const userId = req.user.user_id;
  const { title, message, type } = req.body;

  try {
    const notification = await prisma.notification.create({
      data: {
        userId,
        title,
        message,
        type: "info",
        status: "sending",
        read: false,
        createdAt: new Date(),
      },
    });

    res.status(201).json(notification);
  } catch (error) {
    console.error("Error pushing notification:", error);
    res.status(500).json({ error: "Error pushing notification" });
  }
};

const getNotifications = async (req, res) => {
  const userId = req.user.user_id;

  try {
    const notifications = await prisma.notification.findMany({
      where: { userId },
      orderBy: { createdAt: "desc" },
    });

    res.json(notifications);
  } catch (error) {
    console.error("Error fetching notifications:", error);
    res.status(500).json({ error: "Error fetching notifications" });
  }
};

const updateNotification = async (req, res) => {
  const notificationId = req.params.notificationId;

  try {
    const notification = await prisma.notification.update({
      where: { id: notificationId },
      data: { read: true },
    });

    res.json(notification);
  } catch (error) {
    console.error("Error updating notification:", error);
    res.status(500).json({ error: "Error updating notification" });
  }
};

const sendEmailNotification = async (req, res) => {
  const userID = req.user.user_id;

  const { email, subject, type } = req.body;

  const templateData = {
    name: req.user.name || "Beloved User",
    email: email,
  };

  const templateHTML = getTemplateHTML(type, templateData);

  //store to email
  try {
    await prisma.mail.create({
      data: {
        email: email,
        userId: userID,
        subject: subject,
        template: type,
      },
    });
  } catch (error) {
    console.log("error store email in database : ", error);
  }

  sendMail(email, subject, "", templateHTML)
    .then(() => {
      console.log(`Email sent to ${email} with subject: ${subject}`);
      res.status(200).json({
        message: `Email sent to ${email} with subject: ${subject}`,
      });
    })
    .catch((error) => {
      console.error(`Failed to send email to ${email}:`, error);
      res.status(500).json({
        error: `Failed to send email to ${email}: ${error.message}`,
      });
    });
};

module.exports = {
  pushNotification,
  getNotifications,
  updateNotification,
  sendEmailNotification,
};
