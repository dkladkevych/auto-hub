/**
 * gallery.js
 *
 * Media gallery and lightbox for the listing detail page.
 * Supports both images and MP4 videos with a custom video player.
 *
 * Expected DOM:
 * - <script id="image-data"> with a JSON array of media URLs
 * - #mainViewerImage, #mainImageButton, .thumb-btn
 * - #lightbox, #lightboxContent, #lightboxCloseBtn, #lightboxPrevBtn, #lightboxNextBtn
 *
 * Supports mouse, touch, and keyboard navigation (arrows, Escape).
 */
(function () {
    const imageData = document.getElementById("image-data");
    if (!imageData) return;

    const mediaList = JSON.parse(imageData.textContent);
    let currentIndex = 0;

    const mainImage = document.getElementById("mainViewerImage");
    const mainImageButton = document.getElementById("mainImageButton");
    const thumbButtons = document.querySelectorAll(".thumb-btn");

    const lightbox = document.getElementById("lightbox");
    const lightboxContent = document.getElementById("lightboxContent");
    const lightboxCloseBtn = document.getElementById("lightboxCloseBtn");
    const lightboxPrevBtn = document.getElementById("lightboxPrevBtn");
    const lightboxNextBtn = document.getElementById("lightboxNextBtn");

    if (!mainImage || !lightbox) return;

    function isVideo(url) {
        return typeof url === "string" && url.toLowerCase().endsWith(".mp4");
    }

    function formatTime(seconds) {
        const m = Math.floor(seconds / 60);
        const s = Math.floor(seconds % 60);
        return m + ":" + (s < 10 ? "0" : "") + s;
    }

    // Use container for main viewer so we can replace children easily
    const mainViewerContainer = mainImage.parentElement;

    // Preserve demo ribbon if present
    const demoRibbon = mainViewerContainer ? mainViewerContainer.querySelector(".demo-ribbon") : null;
    if (demoRibbon) demoRibbon.id = "demoRibbon";

    function updateMainViewer(index) {
        currentIndex = index;
        const url = mediaList[currentIndex];
        mainViewerContainer.innerHTML = "";
        if (document.getElementById("demoRibbon")) {
            mainViewerContainer.appendChild(document.getElementById("demoRibbon"));
        }

        if (isVideo(url)) {
            const video = document.createElement("video");
            video.id = "mainViewerImage";
            video.className = "main-viewer-image";
            video.src = url;
            video.muted = true;
            video.playsInline = true;
            video.preload = "metadata";

            const overlay = document.createElement("div");
            overlay.className = "main-video-overlay";

            const playBtn = document.createElement("button");
            playBtn.type = "button";
            playBtn.className = "main-video-play";
            playBtn.setAttribute("aria-label", "Play video");
            playBtn.innerHTML = '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z"/></svg>';

            overlay.appendChild(playBtn);
            mainViewerContainer.appendChild(video);
            mainViewerContainer.appendChild(overlay);
        } else {
            const img = document.createElement("img");
            img.id = "mainViewerImage";
            img.className = "main-viewer-image";
            img.src = url;
            img.alt = "Listing media";
            mainViewerContainer.appendChild(img);
        }

        thumbButtons.forEach((btn, i) => {
            btn.classList.toggle("active", i === currentIndex);
        });
    }

    function createLightboxImage(url) {
        const img = document.createElement("img");
        img.className = "lightbox-image";
        img.src = url;
        img.alt = "Listing media";
        return img;
    }

    function createLightboxVideo(url) {
        const wrap = document.createElement("div");
        wrap.className = "lightbox-video-wrap";

        const video = document.createElement("video");
        video.className = "lightbox-video";
        video.src = url;
        video.playsInline = true;
        video.preload = "auto";
        video.load();

        const overlay = document.createElement("div");
        overlay.className = "video-overlay";

        const bigPlay = document.createElement("button");
        bigPlay.className = "video-big-play";
        bigPlay.innerHTML = '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z"/></svg>';

        const controls = document.createElement("div");
        controls.className = "video-controls";

        const playBtn = document.createElement("button");
        playBtn.className = "vc-play";
        playBtn.innerHTML = '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z"/></svg>';

        const progress = document.createElement("input");
        progress.type = "range";
        progress.className = "vc-progress";
        progress.min = 0;
        progress.max = 100;
        progress.value = 0;

        const time = document.createElement("span");
        time.className = "vc-time";
        time.textContent = "0:00 / 0:00";

        const volWrap = document.createElement("div");
        volWrap.className = "vc-volume-wrap";

        const volIcon = document.createElement("span");
        volIcon.className = "vc-vol-icon";
        const iconImg = document.createElement("img");
        iconImg.src = "/static/images/high_sound.svg";
        iconImg.alt = "Volume";
        volIcon.appendChild(iconImg);

        const volume = document.createElement("input");
        volume.type = "range";
        volume.className = "vc-volume";
        volume.min = 0;
        volume.max = 1;
        volume.step = 0.05;
        volume.value = 1;

        volWrap.appendChild(volIcon);
        volWrap.appendChild(volume);

        controls.appendChild(playBtn);
        controls.appendChild(progress);
        controls.appendChild(time);
        controls.appendChild(volWrap);

        overlay.appendChild(bigPlay);
        wrap.appendChild(video);
        wrap.appendChild(overlay);
        wrap.appendChild(controls);

        function updatePlayState() {
            if (video.paused) {
                playBtn.innerHTML = '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z"/></svg>';
                bigPlay.style.opacity = "1";
            } else {
                playBtn.innerHTML = '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/></svg>';
                bigPlay.style.opacity = "0";
            }
        }

        function togglePlay() {
            if (video.paused) {
                video.play().catch(() => {});
            } else {
                video.pause();
            }
            updatePlayState();
        }

        playBtn.addEventListener("click", togglePlay);
        bigPlay.addEventListener("click", togglePlay);
        video.addEventListener("click", togglePlay);

        video.addEventListener("timeupdate", () => {
            const pct = video.duration ? (video.currentTime / video.duration) * 100 : 0;
            progress.value = pct;
            time.textContent = formatTime(video.currentTime) + " / " + formatTime(video.duration || 0);
        });

        video.addEventListener("ended", () => {
            updatePlayState();
        });

        video.addEventListener("loadedmetadata", () => {
            time.textContent = formatTime(video.currentTime) + " / " + formatTime(video.duration || 0);
        });

        progress.addEventListener("input", () => {
            if (video.duration) {
                video.currentTime = (progress.value / 100) * video.duration;
            }
        });

        volume.addEventListener("input", () => {
            video.volume = volume.value;
            const v = parseFloat(volume.value);
            if (v === 0) {
                iconImg.src = "/static/images/no_sound.svg";
            } else if (v <= 0.5) {
                iconImg.src = "/static/images/low_sound.svg";
            } else {
                iconImg.src = "/static/images/high_sound.svg";
            }
        });

        return wrap;
    }

    function updateNavVisibility() {
        if (mediaList.length <= 1) {
            lightboxPrevBtn.classList.add("hidden");
            lightboxNextBtn.classList.add("hidden");
        } else {
            lightboxPrevBtn.classList.toggle("hidden", currentIndex === 0);
            lightboxNextBtn.classList.toggle("hidden", currentIndex === mediaList.length - 1);
        }
    }

    function openLightbox(index) {
        currentIndex = index;
        updateLightboxMedia();
        updateNavVisibility();
        lightbox.classList.add("is-open");
        document.body.classList.add("lightbox-open");
    }

    function updateLightboxMedia() {
        const url = mediaList[currentIndex];
        lightboxContent.innerHTML = "";
        if (isVideo(url)) {
            lightboxContent.appendChild(createLightboxVideo(url));
        } else {
            lightboxContent.appendChild(createLightboxImage(url));
        }
        updateMainViewer(currentIndex);
        updateNavVisibility();
    }

    function closeLightbox() {
        lightbox.classList.remove("is-open");
        document.body.classList.remove("lightbox-open");
        const video = lightboxContent.querySelector("video");
        if (video) video.pause();
    }

    function prevMedia() {
        currentIndex = (currentIndex - 1 + mediaList.length) % mediaList.length;
        if (lightbox.classList.contains("is-open")) {
            updateLightboxMedia();
        } else {
            updateMainViewer(currentIndex);
        }
    }

    function nextMedia() {
        currentIndex = (currentIndex + 1) % mediaList.length;
        if (lightbox.classList.contains("is-open")) {
            updateLightboxMedia();
        } else {
            updateMainViewer(currentIndex);
        }
    }

    thumbButtons.forEach((btn) => {
        btn.addEventListener("click", () => {
            updateMainViewer(Number(btn.dataset.index));
        });
    });

    if (mainImageButton) {
        mainImageButton.addEventListener("click", () => openLightbox(currentIndex));
    }

    lightbox.addEventListener("click", (event) => {
        if (event.target === lightbox) closeLightbox();
    });

    if (lightboxCloseBtn) lightboxCloseBtn.addEventListener("click", closeLightbox);
    if (lightboxPrevBtn) lightboxPrevBtn.addEventListener("click", prevMedia);
    if (lightboxNextBtn) lightboxNextBtn.addEventListener("click", nextMedia);

    document.addEventListener("keydown", (event) => {
        const isOpen = lightbox.classList.contains("is-open");

        if (event.key === "ArrowLeft") {
            if (isOpen) {
                prevMedia();
            } else {
                currentIndex = (currentIndex - 1 + mediaList.length) % mediaList.length;
                updateMainViewer(currentIndex);
            }
        }

        if (event.key === "ArrowRight") {
            if (isOpen) {
                nextMedia();
            } else {
                currentIndex = (currentIndex + 1) % mediaList.length;
                updateMainViewer(currentIndex);
            }
        }

        if (event.key === "Escape" && isOpen) {
            closeLightbox();
        }
    });
})();
