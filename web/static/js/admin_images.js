(function () {
  window.ImageUploader = function (opts) {
    const container = document.getElementById(opts.containerId);
    const dropZone = document.getElementById(opts.dropZoneId);
    const fileInput = document.getElementById(opts.fileInputId);
    const form = document.querySelector(opts.formSelector);
    const adminPath = opts.adminPath;
    const errorEl = document.getElementById("imageError");

    const MAX_FILE_SIZE = 5 * 1024 * 1024; // 5 MB
    const MAX_IMAGES = 10;
    const ALLOWED_EXTENSIONS = ["jpg", "jpeg", "png", "webp"];

    let images = (opts.initial || []).map(function (url) {
      return { type: "existing", url: url };
    });
    let dragIndex = null;
    let activeSaveMode = null;

    // Ловим клик по кнопкам submit, чтобы знать save_mode
    form.querySelectorAll("button[type='submit']").forEach(function (btn) {
      btn.addEventListener("click", function () {
        activeSaveMode = btn.value;
      });
    });

    function showError(msg) {
      if (!errorEl) return;
      errorEl.textContent = msg;
      errorEl.style.display = "block";
    }

    function clearError() {
      if (!errorEl) return;
      errorEl.textContent = "";
      errorEl.style.display = "none";
    }

    function getExtension(filename) {
      if (!filename || filename.indexOf(".") === -1) return "";
      return filename.split(".").pop().toLowerCase();
    }

    function validateFile(file) {
      const ext = getExtension(file.name);
      if (ALLOWED_EXTENSIONS.indexOf(ext) === -1) {
        return "Only JPG, JPEG, PNG, WEBP images are allowed.";
      }
      if (!file.type || !file.type.startsWith("image/")) {
        return "Only image files are allowed.";
      }
      if (file.size === 0) {
        return "File is empty.";
      }
      if (file.size > MAX_FILE_SIZE) {
        return "Each image must be 5MB or smaller.";
      }
      return null;
    }

    function render() {
      container.innerHTML = "";
      images.forEach(function (img, index) {
        const card = document.createElement("div");
        card.className = "img-thumb-card";
        card.draggable = true;
        card.dataset.index = index;

        const preview = document.createElement("img");
        preview.className = "img-thumb";
        preview.src = img.type === "existing" ? img.url : URL.createObjectURL(img.file);

        const removeBtn = document.createElement("button");
        removeBtn.type = "button";
        removeBtn.className = "img-remove-btn";
        removeBtn.innerHTML = "&times;";
        removeBtn.title = "Remove";
        removeBtn.onclick = function () {
          images.splice(index, 1);
          clearError();
          render();
        };

        card.addEventListener("dragstart", function () {
          dragIndex = index;
          card.classList.add("dragging");
        });
        card.addEventListener("dragend", function () {
          dragIndex = null;
          card.classList.remove("dragging");
        });
        card.addEventListener("dragover", function (e) {
          e.preventDefault();
          if (dragIndex === null || dragIndex === index) return;
          const moved = images.splice(dragIndex, 1)[0];
          images.splice(index, 0, moved);
          dragIndex = index;
          render();
        });

        card.appendChild(preview);
        card.appendChild(removeBtn);
        container.appendChild(card);
      });
    }

    function addFiles(files) {
      clearError();

      const newFiles = [];
      const errors = [];
      for (let i = 0; i < files.length; i++) {
        const f = files[i];
        if (!f.name) continue;

        const error = validateFile(f);
        if (error) {
          errors.push(f.name + ": " + error);
        } else {
          newFiles.push({ type: "new", file: f });
        }
      }

      if (errors.length > 0) {
        showError(errors.join(" "));
        return;
      }

      if (images.length + newFiles.length > MAX_IMAGES) {
        showError("Max " + MAX_IMAGES + " images allowed.");
        return;
      }

      images = images.concat(newFiles);
      render();
    }

    dropZone.addEventListener("click", function () {
      fileInput.click();
    });

    dropZone.addEventListener("dragover", function (e) {
      e.preventDefault();
      dropZone.classList.add("dragover");
    });
    dropZone.addEventListener("dragleave", function () {
      dropZone.classList.remove("dragover");
    });
    dropZone.addEventListener("drop", function (e) {
      e.preventDefault();
      dropZone.classList.remove("dragover");
      if (e.dataTransfer.files.length) {
        addFiles(e.dataTransfer.files);
      }
    });

    fileInput.addEventListener("change", function () {
      addFiles(fileInput.files);
      fileInput.value = "";
    });

    form.addEventListener("submit", async function (e) {
      e.preventDefault();
      const formData = new FormData(form);
      formData.delete("images");
      formData.delete("keep_images");
      formData.delete("save_mode");

      if (activeSaveMode) {
        formData.append("save_mode", activeSaveMode);
      }

      images.forEach(function (img) {
        if (img.type === "new") {
          formData.append("images", img.file);
        } else {
          formData.append("keep_images", img.url);
        }
      });

      const resp = await fetch(form.action || window.location.href, {
        method: "POST",
        body: formData,
        redirect: "follow",
        headers: {
          Accept: "application/json",
          "X-Requested-With": "XMLHttpRequest",
        },
      });

      const isSuccessRedirect =
        resp.url &&
        (resp.url.endsWith("/" + adminPath) ||
          resp.url.endsWith("/" + adminPath + "/"));

      if (isSuccessRedirect) {
        window.location.href = resp.url;
        return;
      }

      const ct = resp.headers.get("content-type") || "";
      if (ct.indexOf("application/json") !== -1) {
        const data = await resp.json();
        if (data.error) {
          showError(data.error);
        }
      } else {
        const html = await resp.text();
        document.open();
        document.write(html);
        document.close();
      }
    });

    render();
  };
})();
