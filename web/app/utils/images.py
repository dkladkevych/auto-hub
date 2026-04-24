"""
Utilities for listing media (images + videos).

- Validation of uploaded files (size, format, MIME type)
- Atomic folder sync via temp directory
- Automatic thumbnail generation for images (Pillow)
- Deletion of media when a listing is removed
- Getting preview URLs and media lists (originals / thumbnails)

Media is stored on the filesystem: data/listings/{id}/01.jpg
Thumbnails: data/listings/{id}/thumb_01.jpg
No DB table is used for media.
"""

import os
import shutil
import subprocess

from PIL import Image

from ..config import Config

ALLOWED_IMAGE_EXTS = {"jpg", "jpeg", "png", "webp", "avif"}
ALLOWED_VIDEO_EXTS = {"mp4"}
MAX_IMAGE_SIZE = 5 * 1024 * 1024       # 5 MB
MAX_VIDEO_SIZE = 15 * 1024 * 1024      # 15 MB
MAX_MEDIA_PER_LISTING = 10

EMPTY_IMAGE = "/static/images/empty.png"

# Max thumbnail size for cards
THUMB_SIZE = (640, 480)


def _get_extension(filename):
    if not filename or "." not in filename:
        return ""
    return filename.rsplit(".", 1)[1].lower()


def _is_video(filename):
    return _get_extension(filename) in ALLOWED_VIDEO_EXTS


def _is_image(filename):
    return _get_extension(filename) in ALLOWED_IMAGE_EXTS


def _listing_dir(listing_id: int):
    return os.path.join(Config.LISTINGS_DIR, str(listing_id))


def validate_media(files):
    """Validate a list of uploaded files. Returns error string or None."""
    valid_files = [f for f in files if f and f.filename]

    if not valid_files:
        return None  # Media is optional

    if len(valid_files) > MAX_MEDIA_PER_LISTING:
        return f"Max {MAX_MEDIA_PER_LISTING} media files allowed."

    for f in valid_files:
        ext = _get_extension(f.filename)
        if ext in ALLOWED_IMAGE_EXTS:
            max_size = MAX_IMAGE_SIZE
            expected_type = "image/"
        elif ext in ALLOWED_VIDEO_EXTS:
            max_size = MAX_VIDEO_SIZE
            expected_type = "video/mp4"
        else:
            return "Only JPG, JPEG, PNG, WEBP, AVIF images and MP4 videos are allowed."

        if f.content_length is not None and f.content_length > max_size:
            return f"Each file must be {max_size // (1024 * 1024)}MB or smaller."

        if expected_type == "video/mp4":
            if not f.content_type or f.content_type != "video/mp4":
                return "Only MP4 video files are allowed."
        else:
            if not f.content_type or not f.content_type.startswith("image/"):
                return "Only image files are allowed."

    return None


def _ensure_max_size(file_obj, max_size):
    """Double-check size via stream seek."""
    file_obj.stream.seek(0, os.SEEK_END)
    size = file_obj.stream.tell()
    file_obj.stream.seek(0)
    if size > max_size:
        raise ValueError("File exceeds maximum allowed size.")


def _filename_for_index(index: int, ext: str):
    """01.jpg, 02.png, 03.mp4, etc."""
    return f"{index:02d}.{ext}"


def _create_thumbnail(src_path: str, dst_path: str):
    """Create a thumbnail via Pillow."""
    try:
        with Image.open(src_path) as img:
            img.thumbnail(THUMB_SIZE)
            img.save(dst_path, quality=85, optimize=True)
    except Exception:
        # Fallback: copy original if thumbnail fails
        shutil.copy2(src_path, dst_path)


def _create_video_thumbnail(src_path: str, dst_path: str):
    """Extract first frame from video via ffmpeg."""
    try:
        subprocess.run(
            ["ffmpeg", "-y", "-i", src_path, "-ss", "00:00:00", "-vframes", "1", dst_path],
            check=True,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
    except Exception:
        pass


def _sync_thumbnails(target_dir: str):
    """Create/update thumbnails for all originals in the folder."""
    for name in os.listdir(target_dir):
        if name.startswith(".") or name.startswith("thumb_"):
            continue
        src = os.path.join(target_dir, name)
        basename = os.path.splitext(name)[0]
        dst = os.path.join(target_dir, f"thumb_{basename}.jpg")
        if os.path.exists(dst):
            continue
        if _is_video(name):
            _create_video_thumbnail(src, dst)
        else:
            _create_thumbnail(src, dst)


def sync_listing_media(listing_id: int, existing_urls, new_files):
    """Sync the listing folder with the given lists.

    existing_urls — list of URLs for already saved media (in desired order).
    new_files — list of FileStorage for new uploads (in desired order).

    Algorithm:
    1. Create a temp folder.
    2. Copy existing files there in order (renaming).
    3. Save new files there (renaming).
    4. Delete old folder and replace with temp.
    5. Generate thumbnails for all originals.

    Returns list of final URLs.
    """
    target_dir = _listing_dir(listing_id)
    temp_dir = target_dir + ".tmp"

    if os.path.exists(temp_dir):
        shutil.rmtree(temp_dir)
    os.makedirs(temp_dir, exist_ok=True)

    saved_paths = []
    idx = 1

    # Existing first (in desired order)
    for url in existing_urls:
        old_name = os.path.basename(url)
        old_path = os.path.join(target_dir, old_name)
        ext = _get_extension(old_name) or "jpg"
        new_name = _filename_for_index(idx, ext)
        new_path = os.path.join(temp_dir, new_name)
        if os.path.exists(old_path):
            shutil.copy2(old_path, new_path)
            saved_paths.append(f"/data/listings/{listing_id}/{new_name}")
            idx += 1

    # Then new uploads
    for f in new_files:
        if not f or not f.filename:
            continue
        ext = _get_extension(f.filename)
        if ext in ALLOWED_IMAGE_EXTS:
            _ensure_max_size(f, MAX_IMAGE_SIZE)
        elif ext in ALLOWED_VIDEO_EXTS:
            _ensure_max_size(f, MAX_VIDEO_SIZE)
        new_name = _filename_for_index(idx, ext)
        new_path = os.path.join(temp_dir, new_name)
        f.save(new_path)
        saved_paths.append(f"/data/listings/{listing_id}/{new_name}")
        idx += 1

    # Atomic replacement
    if os.path.exists(target_dir):
        shutil.rmtree(target_dir)
    os.rename(temp_dir, target_dir)

    # Generate thumbnails
    _sync_thumbnails(target_dir)

    return saved_paths


def delete_listing_images(listing_id: int):
    """Delete the entire media folder for a listing."""
    target_dir = _listing_dir(listing_id)
    if os.path.exists(target_dir):
        shutil.rmtree(target_dir)


def get_listing_media_urls(listing_id: int, thumb: bool = False):
    """Return sorted URLs for a listing's media.

    thumb=False — originals (no thumb_ prefix).
    thumb=True  — thumbnails (with thumb_ prefix). For videos, skips them.
    """
    target_dir = _listing_dir(listing_id)
    if not os.path.exists(target_dir):
        return []

    prefix = "thumb_" if thumb else ""
    files = []
    for name in os.listdir(target_dir):
        if name.startswith("."):
            continue
        if thumb:
            if not name.startswith("thumb_"):
                continue
        else:
            if name.startswith("thumb_"):
                continue
        files.append(name)

    files.sort()
    return [f"/data/listings/{listing_id}/{name}" for name in files]


def get_preview_image(listing_id: int, thumb: bool = False):
    """Return the first media URL or empty placeholder.

    Always picks the first file (01). If a thumbnail exists, returns it;
    otherwise falls back to the original.
    """
    all_urls = get_listing_media_urls(listing_id, thumb=False)
    if not all_urls:
        return EMPTY_IMAGE

    first_url = all_urls[0]
    first_name = os.path.basename(first_url)
    basename = os.path.splitext(first_name)[0]

    if thumb:
        thumb_name = f"thumb_{basename}.jpg"
        thumb_path = os.path.join(_listing_dir(listing_id), thumb_name)
        if os.path.exists(thumb_path):
            return f"/data/listings/{listing_id}/{thumb_name}"

    return first_url


def listing_has_video(listing_id: int) -> bool:
    """Check whether a listing has at least one video file."""
    target_dir = _listing_dir(listing_id)
    if not os.path.exists(target_dir):
        return False
    for name in os.listdir(target_dir):
        if name.startswith(".") or name.startswith("thumb_"):
            continue
        if _is_video(name):
            return True
    return False


# Backwards-compatible aliases
validate_images = validate_media
sync_listing_images = sync_listing_media
get_listing_image_urls = get_listing_media_urls
