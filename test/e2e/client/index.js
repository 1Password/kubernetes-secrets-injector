import http from "http";

const server = http.createServer(() => { });
const PORT = process.env.PORT || 3000;

server.listen(PORT, () => {
    console.log(`Example app listening on port ${PORT}!`);
    console.log(`SECRET: '${process.env.SECRET}'`);
});
