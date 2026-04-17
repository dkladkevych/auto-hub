"""
Утилиты для работы с изображениями объявлений.

- Валидация загружаемых файлов (размер, формат, MIME-тип)
- Синхронизация папки с фото (атомарная замена через temp)
- Автоматическое создание thumbnails при сохранении (Pillow)
- Удаление фото при удалении/редактировании объявления
- Получение URL превью и списка картинок (оригиналы / thumbnails)

Изображения хранятся в файловой системе: data/listings/{id}/01.jpg
Thumbnails: data/listings/{id}/thumb_01.jpg
Таблица БД для изображений не используется.
"""

import os
import shutil

from PIL import Image

from ..config import Config

ALLOWED_EXTENSIONS = {"jpg", "jpeg", "png", "webp"}
MAX_FILE_SIZE = 5 * 1024 * 1024  # 5 MB
MAX_IMAGES_PER_LISTING = 10

EMPTY_IMAGE = "/static/images/empty.png"

# Макс размер thumbnail для карточек на главной
THUMB_SIZE = (640, 480)


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


def _create_thumbnail(src_path: str, dst_path: str):
    """Создаёт thumbnail через Pillow."""
    try:
        with Image.open(src_path) as img:
            img.thumbnail(THUMB_SIZE)
            img.save(dst_path, quality=85, optimize=True)
    except Exception:
        # Если не удалось — копируем оригинал как fallback
        shutil.copy2(src_path, dst_path)


def _sync_thumbnails(target_dir: str):
    """Создаёт/обновляет thumbnails для всех оригиналов в папке."""
    for name in os.listdir(target_dir):
        if name.startswith(".") or name.startswith("thumb_"):
            continue
        src = os.path.join(target_dir, name)
        dst = os.path.join(target_dir, f"thumb_{name}")
        if not os.path.exists(dst):
            _create_thumbnail(src, dst)


def sync_listing_images(listing_id: int, existing_urls, new_files):
    """Синхронизирует папку объявления с заданным списком.

    existing_urls — список URL уже сохранённых картинок (в нужном порядке).
    new_files — список FileStorage новых картинок (в нужном порядке).

    Алгоритм:
    1. Создаёт временную папку.
    2. Копирует туда existing файлы в порядке existing_urls (переименовывая).
    3. Сохраняет туда new_files (переименовывая).
    4. Удаляет старую папку и заменяет на новую.
    5. Генерирует thumbnails для всех оригиналов.

    Возвращает список итоговых URL.
    """
    target_dir = _listing_dir(listing_id)
    temp_dir = target_dir + ".tmp"

    if os.path.exists(temp_dir):
        shutil.rmtree(temp_dir)
    os.makedirs(temp_dir, exist_ok=True)

    saved_paths = []
    idx = 1

    # Сначала существующие (в нужном порядке)
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

    # Генерируем thumbnails
    _sync_thumbnails(target_dir)

    return saved_paths


def delete_listing_images(listing_id: int):
    """Удаляет папку с фотками объявления."""
    target_dir = _listing_dir(listing_id)
    if os.path.exists(target_dir):
        shutil.rmtree(target_dir)


def get_listing_image_urls(listing_id: int, thumb: bool = False):
    """Возвращает отсортированные URL фото объявления из папки.

    thumb=False — оригиналы (без thumb_ prefix).
    thumb=True  — thumbnails (с thumb_ prefix).
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
    """Возвращает URL первой картинки или empty.png."""
    urls = get_listing_image_urls(listing_id, thumb=thumb)
    return urls[0] if urls else EMPTY_IMAGE
