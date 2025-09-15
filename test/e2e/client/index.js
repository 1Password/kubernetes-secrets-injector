import express from "express";
import ejs from "ejs";

const app = express();
app.engine("html", ejs.renderFile);
app.set("view engine", "html");
app.use(express.json()); // for parsing application/json
app.use(express.urlencoded({ extended: true })); // for parsing application/x-www-form-urlencoded

app.listen(5000, () => console.log("Example app listening on port 5000!"));

console.log(process.env);
