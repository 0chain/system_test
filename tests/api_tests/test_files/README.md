# Test Files for Transcoder Tests

This directory contains test files used for testing the 0box transcoder API endpoint (`v2/transcode`).

## Current Test Files

The following sample files are currently included for testing:

- `sample.mp4` - Sample MP4 video file (text placeholder)
- `sample.m3u8` - Sample HLS playlist file (text placeholder)
- `sample.avi` - Sample AVI video file (text placeholder)

## For Production Testing

**Important**: The current files are text placeholders. For comprehensive transcoding tests, you should replace them with actual video files.

### Recommended File Storage Options

1. **Git LFS (Large File Storage)** - Recommended for small to medium files

   ```bash
   # Install Git LFS
   git lfs install

   # Track video files
   git lfs track "*.mp4"
   git lfs track "*.avi"
   git lfs track "*.m3u8"

   # Add and commit
   git add .gitattributes
   git add test_files/
   git commit -m "Add video test files with Git LFS"
   ```

2. **External Storage with Download Scripts**

   - Store files in cloud storage (AWS S3, Google Cloud Storage, etc.)
   - Create download scripts that fetch files before running tests
   - Add files to `.gitignore` to prevent accidental commits

3. **Docker Volume Mounts**
   - Mount a volume containing test files when running tests in Docker
   - Keep files out of the repository entirely

### Recommended Test File Specifications

For comprehensive testing, include files with these characteristics:

#### MP4 Files

- Different resolutions: 480p, 720p, 1080p, 4K
- Different codecs: H.264, H.265/HEVC
- Different bitrates: Low, medium, high
- Different durations: Short (30s), medium (2min), long (10min)

#### HLS Files

- Master playlists with multiple quality variants
- Segment files (.ts files)
- Different adaptive bitrate configurations

#### AVI Files

- Different codecs: XviD, DivX, uncompressed
- Different frame rates: 24fps, 30fps, 60fps

### File Size Considerations

- **Small files** (< 10MB): Good for quick tests, CI/CD
- **Medium files** (10-100MB): Good for performance testing
- **Large files** (> 100MB): Good for stress testing, but may slow down CI/CD

### Security Considerations

- Only use royalty-free or properly licensed content
- Avoid copyrighted material
- Consider using synthetic video generation tools
- Document the source and licensing of all test files

## Test File Management Scripts

Consider creating scripts to manage test files:

```bash
#!/bin/bash
# download_test_files.sh

# Download test files from external storage
aws s3 cp s3://your-bucket/test-files/ ./test_files/ --recursive

# Or use curl for public files
curl -o test_files/sample.mp4 https://example.com/sample.mp4
```

## Integration with CI/CD

For CI/CD pipelines:

1. **Pre-download**: Download test files before running tests
2. **Caching**: Cache downloaded files between runs
3. **Cleanup**: Remove large files after tests complete
4. **Fallback**: Use smaller placeholder files if download fails

Example GitHub Actions workflow:

```yaml
- name: Download test files
  run: |
    ./scripts/download_test_files.sh

- name: Run transcoder tests
  run: |
    go test ./tests/api_tests -v -run Test0BoxTranscoder

- name: Cleanup test files
  run: |
    rm -rf test_files/*.mp4 test_files/*.avi test_files/*.m3u8
```

## Current Implementation

The current test implementation uses text placeholder files to demonstrate the API structure. To use real video files:

1. Replace the placeholder files with actual video files
2. Update the test file paths in `0box_transcoder_test.go` if needed
3. Consider file size and transcoding time in your test timeouts
4. Add appropriate error handling for file operations
