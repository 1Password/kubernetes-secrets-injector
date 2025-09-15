FROM node:alpine
COPY . /app
WORKDIR /app
RUN npm install
ENTRYPOINT ["npm", "start"]
EXPOSE 5000
