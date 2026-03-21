const { exec } = require('child_process');
const path = require('path');

function buildProject(id){
    return new Promise((resolve)=>{
        const child = exec(`cd ${path.join(__dirname, `downloaded/${id}`)} && npm install && npm run build`);
        child.stdout?.on('data', function(data) {
            console.log('stdout: ' + data);
        });
        child.stderr?.on('data', function(data) {
            console.log('stderr: ' + data);
        });

        child.on('close', function(code) {
           resolve("")
        });
    })
}
module.exports = buildProject;