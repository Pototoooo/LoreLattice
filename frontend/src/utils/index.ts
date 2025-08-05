import { MessagePlugin } from "tdesign-vue-next";
export function generateRandomString(length: number) {
  let result = "";
  const characters =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  const charactersLength = characters.length;
  for (let i = 0; i < length; i++) {
    result += characters.charAt(Math.floor(Math.random() * charactersLength));
  }
  return result;
}

export function formatStringDate(date) {
  let data = new Date(date);
  let year = data.getFullYear();
  let month = data.getMonth() + 1;
  let day = data.getDate();
  let hour = data.getHours();
  let minute = data.getMinutes();
  let second = data.getSeconds();
  return (
    year + "-" + month + "-" + day + " " + hour + ":" + minute + ":" + second
  );
}
export function kbFileTypeVerification(file) {
  let validTypes = ["pdf", "txt", "md", "docx", "doc", "jpg", "jpeg", "png"];
  let type = file.name.substring(file.name.lastIndexOf(".") + 1);
  if (!validTypes.includes(type)) {
    MessagePlugin.error("文件类型错误！");
    return true;
  }
  if (
    (type == "pdf" || type == "docx" || type == "doc") &&
    file.size > 31457280
  ) {
    MessagePlugin.error("pdf/doc文件不能超过30M！");
    return true;
  }
  if ((type == "txt" || type == "md") && file.size > 31457280) {
    MessagePlugin.error("txt/md文件不能超过30M！");
    return true;
  }
  return false
}
