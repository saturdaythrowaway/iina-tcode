(()=>{let{core:e,console:t,file:l,mpv:a,utils:o,http:s,event:i,overlay:c,preferences:n}=iina,d="0.0.3";if(l.exists(`@data/tcode-player-${d}`))t.log("tcode-player already exists"),o.exec(`@data/tcode-player-${d}`,["--logfile","/tmp/tcode-player.log","listen","&"]);else{t.log("Downloading tcode-player...");let e=o.resolvePath("@data/");l.list(e,{includeSubDir:!1}).forEach(e=>{e.filename.startsWith("tcode-player-")&&l.delete("@data/"+e.filename)}),s.download("https://github.com/saturdaythrowaway/iina-tcode/releases/latest/download/tcode-player",`@data/tcode-player-${d}`).finally(async()=>{await o.exec("chmod",["a+x",o.resolvePath(`@data/tcode-player-${d}`)]),await o.exec(`@data/tcode-player-${d}`,["--logfile","/tmp/tcode-player.log","listen","&"])})}let p=s.xmlrpc("http://localhost:6800/xmlrpc"),r=e.status.position;function u(e,t=300){let l;return(...a)=>{clearTimeout(l),l=setTimeout(()=>{e.apply(this,a)},t)}}let y=u(()=>{p.call("play",["seek",`${e.status.position}s`])}),m=u(()=>{p.call("pause",["seek",`${e.status.position}s`])});i.on("mpv.unpause",y),i.on("mpv.pause",m),i.on("iina.file-loaded",()=>{let l=decodeURIComponent(e.getRecentDocuments()[0].url).split("/").slice(0,-1).join("/");l.startsWith("file://")?(l=l.slice(7),p.call("load",["folder",l]).then(t=>{e.osd(t)}),p.call("play",["seek","0ms"])):t.log(`dir: ${l}`)}),i.on("iina.window-will-close",()=>{e.osd("closing"),p.call("close",[])}),setInterval(()=>{e.status.position&&e.status.position!==r&&(p.call("seek",["seek",`${e.status.position}s`]),r=e.status.position)},1e3/60)})();
//# sourceMappingURL=index.js.map
