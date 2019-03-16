FROM node:8
WORKDIR /usr/src/app
COPY build/envoy-authz/package*.json ./
RUN npm install
COPY build/envoy-authz/app.js .
CMD [ "npm", "start" ]
