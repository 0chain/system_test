# Small Test Files for Transcoder Tests

This directory contains small test files (< 1MB) for testing the 0box transcoder API endpoint (`v2/transcode`).

## Current Test Files

The following small sample files are currently included for testing:

- `sample.mp4` - Sample MP4 video file (~770KB, W3Schools Big Buck Bunny)
- `sample.avi` - Sample AVI video file (~283KB, FFmpeg tutorial sample)
- `sample.mov` - Sample MOV video file (~283KB, FFmpeg tutorial sample)
- `sample.m3u8` - Sample HLS master playlist (~6.3KB, Apple BipBop with multiple variants)
- `sample_variant.m3u8` - Sample HLS variant playlist (~5.6KB, 480x270 resolution)
- `fileSequence0.ts` through `fileSequence2.ts` - HLS video segments (~280KB each, 6-second duration each)

## File Details

### MP4 File
- **Source**: W3Schools Big Buck Bunny sample
- **Size**: ~770KB
- **Duration**: ~10 seconds
- **Format**: H.264 video with AAC audio

### AVI File
- **Source**: FFmpeg libav tutorial sample
- **Size**: ~283KB
- **Duration**: ~10 seconds
- **Format**: MP4 container (renamed to .avi for testing)

### MOV File
- **Source**: FFmpeg libav tutorial sample
- **Size**: ~283KB
- **Duration**: ~10 seconds
- **Format**: MP4 container (renamed to .mov for testing)

### HLS Files
- **Source**: Apple's BipBop sample stream
- **Master Playlist**: ~6.3KB (references multiple variants)
- **Variant Playlist**: ~5.6KB (480x270 resolution, 30fps)
- **Video Segments**: 3 segments (~280KB each, 6 seconds each)
- **Total Duration**: ~18 seconds
- **Format**: Complete HLS stream with local segments

## Usage

These files are specifically designed for quick transcoding tests where:
- File size is a constraint (< 1MB)
- Quick testing is needed
- Network bandwidth is limited
- CI/CD environments with storage limitations

## Testing

These files can be used to test:
- Basic transcoding functionality
- Different input formats
- Error handling with small files
- Performance with minimal data

## Note

The HLS file is a master playlist only. For complete HLS testing, refer to the main `test_files` directory which contains a complete HLS stream with segments. 