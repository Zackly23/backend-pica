const { PrismaClient } = require("@prisma/client");
const grpc = require("@grpc/grpc-js");
const protoLoader = require("@grpc/proto-loader");
const path = require("path");
const sendMail = require("../mails/utils/mail");
const getTemplateHTML = require("../utils/email.utils");

const prisma = new PrismaClient();
const PROTO_PATH = path.join(__dirname, "../protos/notification.proto");

const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);
const notificationProto = protoDescriptor.notification;

async function sendNotification(call, callback) {
  const { to, subject, type, body, name, metadata } = call.request;

  const templateHTML = getTemplateHTML(type, {
    ...metadata,
    name: name,
  });

  try {
    const user = await prisma.user.findUnique({
      where: { email: to },
    });

    console.log('user : ', user);

    await prisma.mail.create({
      data: {
        email: to,
        userId: user?.id ?? null,
        subject: subject,
        template: templateHTML,
      },
    });

    await sendMail(to, subject, body, templateHTML);

    console.log(`Email sent to ${to} with subject: ${subject}`);
    callback(null, {
      status_code: 200,
      message: "Notification sent successfully",
    });
  } catch (error) {
    console.error(`Failed to send notification to ${to}:`, error);
    callback({
      code: grpc.status.INTERNAL,
      message: "Failed to send notification",
    });
  }
}

function setupGRPCServer() {
  const server = new grpc.Server();
  server.addService(notificationProto.NotificationService.service, {
    SendNotification: sendNotification,
  });

  server.bindAsync(
    "0.0.0.0:50051",
    grpc.ServerCredentials.createInsecure(),
    () => {
      console.log("✅ gRPC server running on port 50051");
      // server.start();
    }
  );
}

module.exports = { setupGRPCServer };
