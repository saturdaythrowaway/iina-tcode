const {
  core,
  console,
  file,
  mpv,
  utils,
  http,
  event,
  overlay,
  standaloneWindow,
  preferences,
} = iina;

let tcodePlayerVersion = "0.0.7";
const tcodePlayerCommand = async () => {
  let loglevel = "info";

  if (tcodePlayerVersion === "dev") {
    loglevel = "debug";
  }

  await utils.exec("killall", [`tcode-player-${tcodePlayerVersion}`]).then(() => {
    utils.exec(`@data/tcode-player-${tcodePlayerVersion}`, [
      "--logfile",
      "/tmp/tcode-player.log",
      "--loglevel",
      loglevel,
      "listen",
      "&",
    ]);
  });
};

if (file.exists(`@data/tcode-player-dev`)) {
  tcodePlayerVersion = "dev";
  tcodePlayerCommand();

} else if (!file.exists(`@data/tcode-player-${tcodePlayerVersion}`)) {
  console.log("Downloading tcode-player...");
  let dir = utils.resolvePath("@data/");
  file
    .list(dir, {
      includeSubDir: false,
    })
    .forEach((f) => {
      if (f.filename.startsWith("tcode-player-")) {
        file.delete("@data/" + f.filename);
      }
    });

  http
    .download(
      "https://github.com/saturdaythrowaway/iina-tcode/releases/latest/download/tcode-player",
      `@data/tcode-player-${tcodePlayerVersion}`,
    )
    .finally(async () => {
      await utils.exec("chmod", [
        "a+x",
        utils.resolvePath(`@data/tcode-player-${tcodePlayerVersion}`),
      ]);
    });
  
  tcodePlayerCommand();
} else {
  tcodePlayerCommand();
  console.log("tcode-player already exists");
}

let rpc = http.xmlrpc("http://localhost:6800/xmlrpc");
let pos = core.status.position;

// preferences.get("")

function debounce(func, timeout = 300) {
  let timer;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => {
      func.apply(this, args);
    }, timeout);
  };
}

const play = debounce(() => {
  console.log("play");
  rpc.call("play", ["seek", `${core.status.position || 0}s`]);
});

const pause = debounce(() => {
  console.log("pause");
  rpc.call("pause", ["seek", `${core.status.position || 0}s`]);
});

// event.on("mpv.unpause", play);
// event.on("mpv.pause", pause);

// event.on("iina.window-loaded", () => {
//   overlay.simpleMode();
//   overlay.setContent(`<img src="file:///tmp/overlay.png">`);
//   overlay.setStyle(`p { color: red }; img {
//     width: 100%;
//     height: 100%;
//     object-fit: contain;
//   }`);
//   overlay.show();
// });

event.on("iina.file-loaded", async () => {
  let path = decodeURIComponent(core.getRecentDocuments()[0].url);
  console.log(path);

  if (path.startsWith("file://")) {
    path = path.slice(7);
    console.log("load");
    rpc.call("load", ["filename", encodeURIComponent(path)]).then((res) => {
      core.osd(res);
      console.log(res);
      
      // rpc.call("render", ["output", "/tmp/overlay.png"]).then((_) => {
      //   overlay.setContent(`<p>Rendering...</p> <img src="file:///tmp/overlay.png" /> <p>${path}</p>`);
      // });
    });
  }
});

event.on("iina.window-will-close", () => {
  core.osd("closing");
  console.log("close");
  rpc.call("close", []);
});

// sync with playback
let wasPlaying = false;
setInterval(() => {
  if (wasPlaying && core.status.paused) {
    pause();
    wasPlaying = false;
  } else if (!wasPlaying && !core.status.paused) {
    play();
    wasPlaying = true;
  }

  if (!core.status.position || core.status.position === pos) return;

  rpc.call("seek", ["seek", `${core.status.position || 0}s`]);
  pos = core.status.position;
}, 1000 / 60);

setInterval(() => {
  rpc.call("set", [
    `min`,
    `${preferences.get("min")}`,
    `max`,
    `${preferences.get("max")}`,
    `offset`,
    `${preferences.get("offset")}ms`,
    `preferAlt`,
    `${preferences.get("preferAlt") ? "true" : "false"}`,
    `preferSoft`,
    `${preferences.get("preferSoft") ? "true" : "false"}`,
    `preferHard`,
    `${preferences.get("preferHard") ? "true" : "false"}`,
  ]);
}, 2000);