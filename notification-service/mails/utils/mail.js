const transporter = require("../config/nodemail.config");

// Kirim email
async function sendMail(to, subject, text, template) {
  
  try {
    const info = await transporter.sendMail({
      from: '"Pictoria App" <pictoria@org.id>', // Nama pengirim
      to, // penerima
      subject, // subjek
      html: template,
    });

    console.log("Email sent:", info.messageId);
  } catch (error) {
    console.error("Error sending email:", error);
  }
}

module.exports = sendMail;
