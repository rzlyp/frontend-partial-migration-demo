# Nuxt 2 needs an older Node (webpack 4). Keep legacy on Node 16.
FROM node:16-alpine
WORKDIR /app
COPY package.json ./
RUN npm install --no-audit --no-fund
COPY . .
RUN npm run build
ENV NODE_ENV=production
EXPOSE 3000
CMD ["npm", "start"]
