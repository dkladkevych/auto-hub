import os
import shutil

from ..config import Config

ALLOWED_EXTENSIONS = {"jpg", "jpeg", "png", "webp"}
MAX_FILE_SIZE = 5 * 1024 * 1024  # 5 MB
MAX_IMAGES_PER_LISTING = 10


def _get_extension(filename):
    if not filename or "." not in filename:
        return ""
    return filename.rsplit(".", 1)[1].lower()


def _listing_dir(listing_id: int):
    return os.path.join(Config.LISTINGS_DIR, str(listing_id))


def validate_images(files):
    """Валидирует список загружаемых файлов. Возвращает ошибку или None."""
    valid_files = [f for f in files if f and f.filename]

    if not valid_files:
        return None  # Фото необязательны

    if len(valid_files) > MAX_IMAGES_PER_LISTING:
        return f"Max {MAX_IMAGES_PER_LISTING} images allowed."

    for f in valid_files:
        ext = _get_extension(f.filename)
        if ext not in ALLOWED_EXTENSIONS:
            return "Only JPG, JPEG, PNG, WEBP images are allowed."

        if f.content_length is not None and f.content_length > MAX_FILE_SIZE:
            return "Each image must be 5MB or smaller."

        if not f.content_type or not f.content_type.startswith("image/"):
            return "Only image files are allowed."

    return None


def _ensure_max_size(file_obj):
    """Дополнительно проверяет размер через stream."""
    file_obj.stream.seek(0, os.SEEK_END)
    size = file_obj.stream.tell()
    file_obj.stream.seek(0)
    if size > MAX_FILE_SIZE:
        raise ValueError("Each image must be 5MB or smaller.")


def _filename_for_index(index: int, ext: str):
    """01.jpg, 02.png и т.д."""
    return f"{index:02d}.{ext}"


def sync_listing_images(listing_id: int, existing_urls, new_files):
    """Синхронизирует папку объявления с заданным списком.

    existing_urls — список URL уже сохранённых картинок (в нужном порядке).
    new_files — список FileStorage новых картинок (в нужном порядке).

    Алгоритм:
    1. Создаёт временную папку.
    2. Копирует туда existing файлы в порядке existing_urls (переименовывая).
    3. Сохраняет туда new_files (переименовывая).
    4. Удаляет старую папку и заменяет на новую.
    5. Обновляет БД (удаляет старые записи, вставляет новые пути).

    Возвращает список итоговых URL.
    """
    from ..db import get_db

    target_dir = _listing_dir(listing_id)
    temp_dir = target_dir + ".tmp"

    if os.path.exists(temp_dir):
        shutil.rmtree(temp_dir)
    os.makedirs(temp_dir, exist_ok=True)

    saved_paths = []
    idx = 1

    # Сначала существующие (в нужном порядке)
    for url in existing_urls:
        # URL вида /data/listings/{id}/01.jpg
        old_name = os.path.basename(url)
        old_path = os.path.join(target_dir, old_name)
        ext = _get_extension(old_name) or "jpg"
        new_name = _filename_for_index(idx, ext)
        new_path = os.path.join(temp_dir, new_name)
        if os.path.exists(old_path):
            shutil.copy2(old_path, new_path)
            saved_paths.append(f"/data/listings/{listing_id}/{new_name}")
            idx += 1

    # Затем новые
    for f in new_files:
        if not f or not f.filename:
            continue
        _ensure_max_size(f)
        ext = _get_extension(f.filename)
        new_name = _filename_for_index(idx, ext)
        new_path = os.path.join(temp_dir, new_name)
        f.save(new_path)
        saved_paths.append(f"/data/listings/{listing_id}/{new_name}")
        idx += 1

    # Атомарная замена папки
    if os.path.exists(target_dir):
        shutil.rmtree(target_dir)
    os.rename(temp_dir, target_dir)

    # Обновляем БД
    db = get_db()
    db.execute("DELETE FROM listing_images WHERE listing_id = ?", (listing_id,))
    for path in saved_paths:
        db.execute(
            "INSERT INTO listing_images (listing_id, image_url) VALUES (?, ?)",
            (listing_id, path),
        )
    db.commit()
    db.close()

    return saved_paths


def delete_listing_images(listing_id: int):
    """Удаляет папку с фотками объявления и записи в БД."""
    from ..db import get_db

    target_dir = _listing_dir(listing_id)
    if os.path.exists(target_dir):
        shutil.rmtree(target_dir)

    db = get_db()
    db.execute("DELETE FROM listing_images WHERE listing_id = ?", (listing_id,))
    db.commit()
    db.close()


def get_listing_image_urls(listing_id: int):
    """Возвращает отсортированные URL фото объявления из папки."""
    target_dir = _listing_dir(listing_id)
    if not os.path.exists(target_dir):
        return []

    files = []
    for name in os.listdir(target_dir):
        if name.startswith("."):
            continue
        files.append(name)

    files.sort()
    return [f"/data/listings/{listing_id}/{name}" for name in files]


def get_preview_image(listing_id: int):
    """Возвращает URL первой картинки или None."""
    urls = get_listing_image_urls(listing_id)
    return urls[0] if urls else None
