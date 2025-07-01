FROM node:18-alpine

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm install

COPY . .

RUN npx prisma generate

EXPOSE 3000
EXPOSE 5001

CMD ["node", "app.js"]
