
# Every Anime Frame In Order


## 📁 Project Structure

```
.
├── main.go               # Main program file
├── config.json           # Configuration file
├── framelogs.json        # Tracks upload progress
└── frame/
    ├── S1/
    │   └── eps1/
    │       ├── frame_1.png
    │       ├── frame_2.png
    │       └── ...
    └── S2/
        └── eps3/
            └── ...
```

---

## ⚙️ Configuration

Create a file named `config.json`:

```json
{
  "access_token": "YOUR_PAGE_ACCESS_TOKEN",
  "id": "YOUR_PAGE_ID",
  "max_eps": 5,
  "max_season": 3
}
```

- `access_token`: Your **Facebook Page Access Token**
- `id`: Your **Facebook Page ID**
- `max_eps`: Maximum episodes per season to consider
- `max_season`: Maximum number of seasons

---

## 📝 Upload Log File

Create a file named `framelogs.json`:

```json
{
  "frame": 1,
  "eps": 1,
  "season": 1,
  "is_random": false
}
```

- This file tracks the last uploaded frame to allow resuming the upload if interrupted.

---

## 🚀 How It Works

### 🔁 Sequential Mode

- Begins from `frame.Season`, `frame.Eps`, and `frame.Frame`
- Iterates through all frames in sequence
- Sleeps for **3 hours** (`10800 seconds`) between uploads
- Switches to **random mode** once complete

### 🎲 Random Mode

- Randomly selects:
  - Season: `1 ~ MaxSeason`
  - Episode: `1 ~ MaxEps`
  - Frame: `1 ~ max frames in folder`
- Verifies that selected folder and frame exist before attempting upload
- Sleeps for **3 hours** between uploads (adjust as needed)

---

## 🧪 Running the Program

```bash
go run main.go
```

Ensure that:

- Your `frame/` folder is properly structured.
- PNG images are named like `frame_1.png`, `frame_2.png`, etc.
- You have a valid Facebook **Page Access Token** with permission to publish photos.

---

## 📌 Notes

- The app uses the Facebook Graph API v23.0.
- Frame uploads are sent as `multipart/form-data` with:
  - `source`: the image file
  - `caption`: frame metadata
  - `access_token`: the page token
- The app gracefully skips empty or missing directories.
- All progress is saved to `framelogs.json`.

---
