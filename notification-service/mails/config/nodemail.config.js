const nodemailer = require("nodemailer");

// Looking to send emails in production? Check out our Email API/SMTP product!
var transporter = nodemailer.createTransport({
  host: "sandbox.smtp.mailtrap.io",
  port: 2525,
  auth: {
    user: "fc9ed2276ab7a6",
    pass: "782191b198649e"
  }
});

module.exports = transporter;
