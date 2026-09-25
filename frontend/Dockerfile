FROM node:20-alpine

ARG API_PROXY_TARGET=http://backend:8080
ENV API_PROXY_TARGET=$API_PROXY_TARGET

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm install

COPY . .

# Next.js telemetry disable during build
ENV NEXT_TELEMETRY_DISABLED 1

RUN npm run build

EXPOSE 3000

CMD ["npm", "start"]
