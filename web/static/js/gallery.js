/**
 * gallery.js
 *
 * Галерея изображений и лайтбокс для детальной страницы объявления.
 * Связан с шаблоном public/listing.html.
 *
 * Ожидает в DOM:
 * - <script id="image-data"> с JSON-массивом URL картинок
 * - #mainViewerImage, #mainImageButton, .thumb-btn
 * - #lightbox, #lightbox-image, #lightboxCloseBtn, #lightboxPrevBtn, #lightboxNextBtn
 *
 * Поддерживает навигацию мышью, тачем и клавиатурой (стрелки, Escape).
 */
(function () {
    const imageData = document.getElementById("image-data");
    if (!imageData) return;

    const imageList = JSON.parse(imageData.textContent);
    let currentImageIndex = 0;

    const mainImage = document.getElementById("mainViewerImage");
    const mainImageButton = document.getElementById("mainImageButton");
    const thumbButtons = document.querySelectorAll(".thumb-btn");

    const lightbox = document.getElementById("lightbox");
    const lightboxImage = document.getElementById("lightbox-image");
    const lightboxCloseBtn = document.getElementById("lightboxCloseBtn");
    const lightboxPrevBtn = document.getElementById("lightboxPrevBtn");
    const lightboxNextBtn = document.getElementById("lightboxNextBtn");

    if (!mainImage || !lightbox) return;

    function setMainImage(index) {
        currentImageIndex = index;
        mainImage.src = imageList[currentImageIndex];
        thumbButtons.forEach((btn, i) => {
            btn.classList.toggle("active", i === currentImageIndex);
        });
    }

    function openLightbox(index) {
        currentImageIndex = index;
        lightboxImage.src = imageList[currentImageIndex];
        lightbox.classList.add("is-open");
        document.body.classList.add("lightbox-open");
    }

    function updateLightboxImage() {
        lightboxImage.src = imageList[currentImageIndex];
        setMainImage(currentImageIndex);
    }

    function closeLightbox() {
        lightbox.classList.remove("is-open");
        document.body.classList.remove("lightbox-open");
    }

    function prevImage() {
        currentImageIndex = (currentImageIndex - 1 + imageList.length) % imageList.length;
        updateLightboxImage();
    }

    function nextImage() {
        currentImageIndex = (currentImageIndex + 1) % imageList.length;
        updateLightboxImage();
    }

    thumbButtons.forEach((btn) => {
        btn.addEventListener("click", () => {
            setMainImage(Number(btn.dataset.index));
        });
    });

    if (mainImageButton) {
        mainImageButton.addEventListener("click", () => openLightbox(currentImageIndex));
    }

    lightbox.addEventListener("click", (event) => {
        if (event.target === lightbox) closeLightbox();
    });

    if (lightboxCloseBtn) lightboxCloseBtn.addEventListener("click", closeLightbox);
    if (lightboxPrevBtn) lightboxPrevBtn.addEventListener("click", prevImage);
    if (lightboxNextBtn) lightboxNextBtn.addEventListener("click", nextImage);

    document.addEventListener("keydown", (event) => {
        const isOpen = lightbox.classList.contains("is-open");

        if (event.key === "ArrowLeft") {
            if (isOpen) {
                prevImage();
            } else {
                currentImageIndex = (currentImageIndex - 1 + imageList.length) % imageList.length;
                setMainImage(currentImageIndex);
            }
        }

        if (event.key === "ArrowRight") {
            if (isOpen) {
                nextImage();
            } else {
                currentImageIndex = (currentImageIndex + 1) % imageList.length;
                setMainImage(currentImageIndex);
            }
        }

        if (event.key === "Escape" && isOpen) {
            closeLightbox();
        }
    });
})();
