package mixing

import (
	"encoder/command"
	"strings"
	"testing"
)

func TestNewMixingBuilder(t *testing.T) {
	builder := NewMixingBuilder("/input/video.mp4", "/output/mixed.mp4")

	if builder.videoInput != "/input/video.mp4" {
		t.Error("Expected videoInput to be set")
	}
	if builder.outputPath != "/output/mixed.mp4" {
		t.Error("Expected outputPath to be set")
	}
	if !builder.copyVideo {
		t.Error("Expected copyVideo to be true by default")
	}
	if !builder.copyAudio {
		t.Error("Expected copyAudio to be true by default")
	}
	if builder.priority != command.PriorityNormal {
		t.Error("Expected default priority to be PriorityNormal")
	}
}

func TestMixingBuilder_AddAudioTrack(t *testing.T) {
	builder := NewMixingBuilder("/input/video.mp4", "/output/mixed.mp4")
	builder.AddAudioTrack("/input/audio1.aac").
		AddAudioTrack("/input/audio2.mp3")

	if len(builder.audioInputs) != 2 {
		t.Errorf("Expected 2 audio tracks, got %d", len(builder.audioInputs))
	}

	if builder.audioInputs[0] != "/input/audio1.aac" {
		t.Error("First audio track not set correctly")
	}
	if builder.audioInputs[1] != "/input/audio2.mp3" {
		t.Error("Second audio track not set correctly")
	}
}

func TestMixingBuilder_AddSubtitleTrack(t *testing.T) {
	builder := NewMixingBuilder("/input/video.mp4", "/output/mixed.mp4")
	builder.AddSubtitleTrack("/input/subtitles.srt")

	if builder.subtitleInput != "/input/subtitles.srt" {
		t.Error("Subtitle track not set correctly")
	}
}

func TestMixingBuilder_StreamCopying(t *testing.T) {
	builder := NewMixingBuilder("/input/video.mp4", "/output/mixed.mp4")
	builder.AddAudioTrack("/input/audio.aac")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Should copy video and audio by default
	if !strings.Contains(argsStr, "-c:v copy") {
		t.Error("Expected video stream copy")
	}
	if !strings.Contains(argsStr, "-c:a copy") {
		t.Error("Expected audio stream copy")
	}
}

func TestMixingBuilder_ReEncoding(t *testing.T) {
	builder := NewMixingBuilder("/input/video.mp4", "/output/mixed.mp4")
	builder.AddAudioTrack("/input/audio.flac").
		SetCopyVideo(false).
		SetVideoCodec("libx264").
		SetVideoBitrate("5M").
		SetCopyAudio(false).
		SetAudioCodec("aac").
		SetAudioBitrate("192k")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Should re-encode with specified codecs
	if !strings.Contains(argsStr, "-c:v libx264") {
		t.Error("Expected video codec libx264")
	}
	if !strings.Contains(argsStr, "-b:v 5M") {
		t.Error("Expected video bitrate 5M")
	}
	if !strings.Contains(argsStr, "-c:a aac") {
		t.Error("Expected audio codec aac")
	}
	if !strings.Contains(argsStr, "-b:a 192k") {
		t.Error("Expected audio bitrate 192k")
	}
}

func TestMixingBuilder_StreamMapping(t *testing.T) {
	builder := NewMixingBuilder("/input/video.mp4", "/output/mixed.mp4")
	builder.AddAudioTrack("/input/audio.aac")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Default mapping: video from input 0, audio from input 1
	if !strings.Contains(argsStr, "-map 0:v") {
		t.Error("Expected video mapping from first input")
	}
	if !strings.Contains(argsStr, "-map 1:a") {
		t.Error("Expected audio mapping from second input")
	}
}

func TestMixingBuilder_CustomStreamMapping(t *testing.T) {
	builder := NewMixingBuilder("/input/video.mp4", "/output/mixed.mp4")
	builder.AddAudioTrack("/input/audio.aac").
		MapStream("0:v:0").
		MapStream("1:a:0")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "-map 0:v:0") {
		t.Error("Expected custom video mapping")
	}
	if !strings.Contains(argsStr, "-map 1:a:0") {
		t.Error("Expected custom audio mapping")
	}
}

func TestMixingBuilder_Metadata(t *testing.T) {
	builder := NewMixingBuilder("/input/video.mp4", "/output/mixed.mp4")
	builder.AddMetadata("title", "My Video").
		AddMetadata("author", "John Doe").
		AddMetadata("year", "2025")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "-metadata title=My Video") {
		t.Error("Expected title metadata")
	}
	if !strings.Contains(argsStr, "-metadata author=John Doe") {
		t.Error("Expected author metadata")
	}
	if !strings.Contains(argsStr, "-metadata year=2025") {
		t.Error("Expected year metadata")
	}
}

func TestMixingBuilder_MultipleAudioTracks(t *testing.T) {
	builder := NewMixingBuilder("/input/video.mp4", "/output/mixed.mp4")
	builder.AddAudioTrack("/input/english.aac").
		AddAudioTrack("/input/spanish.aac").
		AddAudioTrack("/input/french.aac")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Should have 3 audio inputs
	if !strings.Contains(argsStr, "-i /input/english.aac") {
		t.Error("Expected first audio input")
	}
	if !strings.Contains(argsStr, "-i /input/spanish.aac") {
		t.Error("Expected second audio input")
	}
	if !strings.Contains(argsStr, "-i /input/french.aac") {
		t.Error("Expected third audio input")
	}

	// Should map all audio tracks
	if !strings.Contains(argsStr, "-map 1:a") {
		t.Error("Expected mapping for first audio track")
	}
	if !strings.Contains(argsStr, "-map 2:a") {
		t.Error("Expected mapping for second audio track")
	}
	if !strings.Contains(argsStr, "-map 3:a") {
		t.Error("Expected mapping for third audio track")
	}
}

func TestMixingBuilder_WithSubtitles(t *testing.T) {
	builder := NewMixingBuilder("/input/video.mp4", "/output/mixed.mp4")
	builder.AddAudioTrack("/input/audio.aac").
		AddSubtitleTrack("/input/subtitles.srt")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Should include subtitle input
	if !strings.Contains(argsStr, "-i /input/subtitles.srt") {
		t.Error("Expected subtitle input")
	}

	// Should map subtitle stream
	if !strings.Contains(argsStr, "-map 2:s") {
		t.Error("Expected subtitle stream mapping")
	}

	// Should copy subtitle codec
	if !strings.Contains(argsStr, "-c:s copy") {
		t.Error("Expected subtitle codec copy")
	}
}

func TestMixingBuilder_ExtraArgs(t *testing.T) {
	builder := NewMixingBuilder("/input/video.mp4", "/output/mixed.mp4")
	builder.AddExtraArgs("-movflags", "+faststart", "-threads", "4")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "-movflags +faststart") {
		t.Error("Expected movflags argument")
	}
	if !strings.Contains(argsStr, "-threads 4") {
		t.Error("Expected threads argument")
	}
}

func TestMixingBuilder_DryRun(t *testing.T) {
	builder := NewMixingBuilder("/input/video.mp4", "/output/mixed.mp4")
	builder.AddAudioTrack("/input/audio.aac")

	cmd, err := builder.DryRun()
	if err != nil {
		t.Fatalf("DryRun failed: %v", err)
	}

	if !strings.HasPrefix(cmd, "ffmpeg") {
		t.Error("Expected command to start with 'ffmpeg'")
	}

	if !strings.Contains(cmd, "/input/video.mp4") {
		t.Error("Expected video input in command")
	}

	if !strings.Contains(cmd, "/input/audio.aac") {
		t.Error("Expected audio input in command")
	}

	if !strings.Contains(cmd, "/output/mixed.mp4") {
		t.Error("Expected output path in command")
	}
}

func TestMixingBuilder_CommandInterface(t *testing.T) {
	builder := NewMixingBuilder("/input/video.mp4", "/output/mixed.mp4")
	builder.SetPriority(10)

	if builder.GetPriority() != 10 {
		t.Errorf("Expected priority 10, got %d", builder.GetPriority())
	}

	if builder.GetTaskType() != command.TaskTypeMixing {
		t.Errorf("Expected task type 'mixing', got '%s'", builder.GetTaskType())
	}

	if builder.GetInputPath() != "/input/video.mp4" {
		t.Errorf("Expected input path '/input/video.mp4', got '%s'", builder.GetInputPath())
	}

	if builder.GetOutputPath() != "/output/mixed.mp4" {
		t.Errorf("Expected output path '/output/mixed.mp4', got '%s'", builder.GetOutputPath())
	}
}

func TestMixingBuilder_FluentAPI(t *testing.T) {
	builder := NewMixingBuilder("/input/video.mp4", "/output/mixed.mp4").
		AddAudioTrack("/input/audio1.aac").
		AddAudioTrack("/input/audio2.mp3").
		AddSubtitleTrack("/input/subs.srt").
		SetCopyVideo(true).
		SetCopyAudio(true).
		AddMetadata("title", "Test Video").
		AddExtraArgs("-threads", "8")

	if len(builder.audioInputs) != 2 {
		t.Error("Fluent API failed to add audio tracks")
	}

	if builder.subtitleInput != "/input/subs.srt" {
		t.Error("Fluent API failed to set subtitle")
	}

	if !builder.copyVideo || !builder.copyAudio {
		t.Error("Fluent API failed to set copy flags")
	}
}

func TestMixingBuilder_ComplexMixing(t *testing.T) {
	// Real-world scenario: Mix video with multiple audio tracks and subtitles
	builder := NewMixingBuilder("/input/video.mkv", "/output/final.mkv")
	builder.AddAudioTrack("/input/english.aac").
		AddAudioTrack("/input/commentary.mp3").
		AddSubtitleTrack("/input/english.srt").
		SetCopyVideo(true).
		SetCopyAudio(true).
		AddMetadata("title", "My Movie").
		AddMetadata("year", "2025").
		AddExtraArgs("-movflags", "+faststart")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Verify all components are present
	expectedParts := []string{
		"-i /input/video.mkv",
		"-i /input/english.aac",
		"-i /input/commentary.mp3",
		"-i /input/english.srt",
		"-map 0:v",
		"-map 1:a",
		"-map 2:a",
		"-map 3:s",
		"-c:v copy",
		"-c:a copy",
		"-c:s copy",
		"-metadata title=My Movie",
		"-metadata year=2025",
		"-movflags +faststart",
		"/output/final.mkv",
	}

	for _, part := range expectedParts {
		if !strings.Contains(argsStr, part) {
			t.Errorf("Expected command to contain: %s", part)
		}
	}
}
