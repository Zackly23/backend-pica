const path = require("path");
const fs = require("fs");

function getTemplateHTML(type, variables = {}) {
  let templateFile = "";

  switch (type) {
    case "password-reset":
      templateFile = "reset.password.html";
      break;
    case "deactivate-account":
      templateFile = "deactivate.account.html";
      break;
    case "two-factor-auth":
      templateFile = "twofactor.auth.html";
      break;
    case "subscription":
      templateFile = "subscription.html";
      break;
    case "subscription-due":
      templateFile = "subscription.due.html";
      break;
    default:
      throw new Error("Unknown template type");
  }

  const filePath = path.join(__dirname, "../mails/templates", templateFile);

  if (!fs.existsSync(filePath)) {
    throw new Error(`Template file not found: ${filePath}`);
  }

  let html = fs.readFileSync(filePath, "utf-8");

  // Inject dynamic variables like {{name}}, {{link}}, etc.
  for (const key in variables) {
    const value = variables[key];
    html = html.replace(new RegExp(`{{${key}}}`, "g"), value);
  }

  return html;
}

module.exports = getTemplateHTML;
