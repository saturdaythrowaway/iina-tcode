(()=>{let{core:e,console:t,file:l,mpv:a,utils:o,http:s,event:i,overlay:d,preferences:c}=iina,n="0.0.4",p=()=>{let e="info";c.get("tcode-player-loglevel")&&(e=c.get("tcode-player-loglevel")),"dev"===n&&(e="debug"),o.exec("killall",[`tcode-player-${n}`]).then(()=>{o.exec(`@data/tcode-player-${n}`,["--logfile","/tmp/tcode-player.log","--loglevel",e,"listen","&"])})};if(l.exists("@data/tcode-player-dev")&&(n="dev"),l.exists(`@data/tcode-player-${n}`))t.log("tcode-player already exists"),p();else{t.log("Downloading tcode-player...");let e=o.resolvePath("@data/");l.list(e,{includeSubDir:!1}).forEach(e=>{e.filename.startsWith("tcode-player-")&&l.delete("@data/"+e.filename)}),s.download("https://github.com/saturdaythrowaway/iina-tcode/releases/latest/download/tcode-player",`@data/tcode-player-${n}`).finally(async()=>{await o.exec("chmod",["a+x",o.resolvePath(`@data/tcode-player-${n}`)]),await p()})}let r=s.xmlrpc("http://localhost:6800/xmlrpc"),u=e.status.position;function y(e,t=300){let l;return(...a)=>{clearTimeout(l),l=setTimeout(()=>{e.apply(this,a)},t)}}let g=y(()=>{t.log("play"),r.call("play",["seek",`${e.status.position||0}s`])}),h=y(()=>{t.log("pause"),r.call("pause",["seek",`${e.status.position||0}s`])});i.on("iina.file-loaded",()=>{let l=decodeURIComponent(e.getRecentDocuments()[0].url).split("/").slice(0,-1).join("/");l.startsWith("file://")?(l=l.slice(7),t.log("load"),r.call("load",["folder",l]).then(t=>{e.osd(t)})):t.log(`dir: ${l}`)}),i.on("iina.window-will-close",()=>{e.osd("closing"),t.log("close"),r.call("close",[])});let f=!1;setInterval(()=>{f&&e.status.paused?(h(),f=!1):f||e.status.paused||(g(),f=!0),e.status.position&&e.status.position!==u&&(r.call("seek",["seek",`${e.status.position||0}s`]),u=e.status.position)},1e3/60)})();
//# sourceMappingURL=index.js.map
