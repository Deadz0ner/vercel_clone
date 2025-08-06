const max_Len=5;
function generate(){
    let id="";
    const chars="123456789qwertyuiopasdfghjklzxcvbnm";
    for(let i=0;i<max_Len;i++){
        id+=chars[Math.floor(Math.random()*chars.length)];
    }
    return id;
}
module.exports = { generate }
