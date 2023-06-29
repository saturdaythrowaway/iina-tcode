const { core, console, file, mpv, utils, http, event, overlay, preferences } = iina;

const tcodePlayerVersion = "0.0.1";

if (!file.exists(`@data/tcode-player-${tcodePlayerVersion}`)) {
  console.log("Downloading tcode-player...");
  let dir = utils.resolvePath("@data/");
  file.list(dir, {
    includeSubDir: false,
  }).forEach((f) => {
    if (f.filename.startsWith("tcode-player-")) {
      file.delete("@data/" + f.filename);
    }
  });

  http.download("https://github.com/saturdaythrowaway/iina-tcode/releases/latest/download/tcode-player", `@data/tcode-player-${tcodePlayerVersion}`).finally(
    async () => {
      await utils.exec("chmod", ["a+x", utils.resolvePath(`@data/tcode-player-${tcodePlayerVersion}`)])
      await utils.exec(`@data/tcode-player-${tcodePlayerVersion}`, ["--logfile", "/tmp/tcode-player.log", "listen", "&"]);
    }
  )
  
} else {
  console.log("tcode-player already exists")
  utils.exec(`@data/tcode-player-${tcodePlayerVersion}`, ["--logfile", "/tmp/tcode-player.log", "listen", "&"]);
}

let rpc = http.xmlrpc("http://localhost:6800/xmlrpc");
let pos = core.status.position;

// preferences.get("")

function debounce(func, timeout = 300){
  let timer;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => { func.apply(this, args); }, timeout);
  };
}

const play = debounce(() => {
  rpc.call("play", ["seek", `${core.status.position}s`]);
})

const pause = debounce(() => {
  rpc.call("pause", ["seek", `${core.status.position}s`]);
})

event.on("mpv.unpause", play);
event.on("mpv.pause", pause);


event.on("iina.file-loaded", () => {
  let dir = decodeURIComponent(core.getRecentDocuments()[0].url)
    .split("/")
    .slice(0, -1)
    .join("/");
  if (dir.startsWith("file://")) {
    dir = dir.slice(7);
    rpc.call("load", ["folder", dir]).then((res) => {
      core.osd(res)
    })

    // rpc.call("render", ["output", "/tmp/overlay.png"])
    // overlay.simpleMode();
    // overlay.setContent("<img src='file:///tmp/overlay.png' />")
    // overlay.show();
    
    rpc.call("play", ["seek", "0ms"]);
  } else {
    console.log(`dir: ${dir}`);
  }
});

event.on("iina.window-will-close", () => {
  core.osd("closing")
  rpc.call("close", []);
})

// sync with playback
setInterval(() => {
  if (!core.status.position || core.status.position === pos) return;
  rpc.call("seek", ["seek", `${core.status.position}s`]);
  pos = core.status.position;
}, 1000/60);