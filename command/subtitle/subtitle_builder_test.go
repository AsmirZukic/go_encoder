package subtitle

import (
	"encoder/command"
	"strings"
	"testing"
)

func TestNewSubtitleBuilder(t *testing.T) {
	builder := NewSubtitleBuilder("/input/video.mp4", "/output/subtitles.srt")

	if builder.inputPath != "/input/video.mp4" {
		t.Error("Expected inputPath to be set")
	}
	if builder.outputPath != "/output/subtitles.srt" {
		t.Error("Expected outputPath to be set")
	}
	if builder.streamIndex != -1 {
		t.Error("Expected streamIndex to be -1 (auto) by default")
	}
	if builder.priority != command.PriorityNormal {
		t.Error("Expected default priority to be PriorityNormal")
	}
}

func TestSubtitleBuilder_SetStreamIndex(t *testing.T) {
	builder := NewSubtitleBuilder("/input/video.mp4", "/output/subs.srt")
	builder.SetStreamIndex(2)

	if builder.streamIndex != 2 {
		t.Errorf("Expected streamIndex 2, got %d", builder.streamIndex)
	}

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "-map 0:s:2") {
		t.Error("Expected mapping to subtitle stream 2")
	}
}

func TestSubtitleBuilder_SetFormat(t *testing.T) {
	builder := NewSubtitleBuilder("/input/video.mp4", "/output/subs.srt")
	builder.SetFormat(FormatSRT)

	if builder.format != FormatSRT {
		t.Error("Expected format to be SRT")
	}

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "-c:s srt") {
		t.Error("Expected SRT codec")
	}
}

func TestSubtitleBuilder_SetLanguage(t *testing.T) {
	builder := NewSubtitleBuilder("/input/video.mp4", "/output/subs.srt")
	builder.SetLanguage("eng")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "-map 0:m:language:eng") {
		t.Error("Expected language filter for English")
	}
}

func TestSubtitleBuilder_BasicExtraction(t *testing.T) {
	builder := NewSubtitleBuilder("/input/video.mp4", "/output/subs.srt")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Should extract first subtitle stream by default
	if !strings.Contains(argsStr, "-map 0:s:0") {
		t.Error("Expected mapping to first subtitle stream")
	}

	// Should copy by default
	if !strings.Contains(argsStr, "-c:s copy") {
		t.Error("Expected subtitle stream copy")
	}
}

func TestSubtitleBuilder_ConvertFormat(t *testing.T) {
	builder := NewSubtitleBuilder("/input/video.mkv", "/output/subs.vtt")
	builder.ConvertFormat(FormatVTT)

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "-c:s vtt") {
		t.Error("Expected VTT codec for conversion")
	}
}

func TestSubtitleBuilder_BurnIntoVideo(t *testing.T) {
	builder := NewSubtitleBuilder("/input/video.mp4", "/output/video_with_subs.mp4")
	builder.BurnIntoVideo("/input/subtitles.srt")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Should use subtitles filter
	if !strings.Contains(argsStr, "subtitles=") {
		t.Error("Expected subtitles filter for burn-in")
	}

	// Should copy audio
	if !strings.Contains(argsStr, "-c:a copy") {
		t.Error("Expected audio copy during burn-in")
	}

	// Should have subtitle file path
	if !strings.Contains(argsStr, "/input/subtitles.srt") {
		t.Error("Expected subtitle file path in filter")
	}
}

func TestSubtitleBuilder_BurnInASS(t *testing.T) {
	builder := NewSubtitleBuilder("/input/video.mp4", "/output/video_with_subs.mp4")
	builder.BurnIntoVideo("/input/subtitles.ass")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Should use ASS filter for .ass files
	if !strings.Contains(argsStr, "ass=") {
		t.Error("Expected ass filter for .ass files")
	}
}

func TestSubtitleBuilder_BurnInWithStyle(t *testing.T) {
	builder := NewSubtitleBuilder("/input/video.mp4", "/output/video_with_subs.mp4")
	builder.BurnIntoVideo("/input/subtitles.srt").
		SetBurnInStyle("FontName=Arial,FontSize=24")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "force_style='FontName=Arial,FontSize=24'") {
		t.Error("Expected burn-in style in filter")
	}
}

func TestSubtitleBuilder_ExtraArgs(t *testing.T) {
	builder := NewSubtitleBuilder("/input/video.mp4", "/output/subs.srt")
	builder.AddExtraArgs("-threads", "4", "-loglevel", "info")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "-threads 4") {
		t.Error("Expected threads argument")
	}
	if !strings.Contains(argsStr, "-loglevel info") {
		t.Error("Expected loglevel argument")
	}
}

func TestSubtitleBuilder_DryRun(t *testing.T) {
	builder := NewSubtitleBuilder("/input/video.mp4", "/output/subs.srt")
	builder.SetStreamIndex(1).SetFormat(FormatSRT)

	cmd, err := builder.DryRun()
	if err != nil {
		t.Fatalf("DryRun failed: %v", err)
	}

	if !strings.HasPrefix(cmd, "ffmpeg") {
		t.Error("Expected command to start with 'ffmpeg'")
	}

	if !strings.Contains(cmd, "/input/video.mp4") {
		t.Error("Expected input path in command")
	}

	if !strings.Contains(cmd, "/output/subs.srt") {
		t.Error("Expected output path in command")
	}
}

func TestSubtitleBuilder_CommandInterface(t *testing.T) {
	builder := NewSubtitleBuilder("/input/video.mp4", "/output/subs.srt")
	builder.SetPriority(8)

	if builder.GetPriority() != 8 {
		t.Errorf("Expected priority 8, got %d", builder.GetPriority())
	}

	if builder.GetTaskType() != command.TaskTypeSubtitle {
		t.Errorf("Expected task type 'subtitle', got '%s'", builder.GetTaskType())
	}

	if builder.GetInputPath() != "/input/video.mp4" {
		t.Errorf("Expected input path '/input/video.mp4', got '%s'", builder.GetInputPath())
	}

	if builder.GetOutputPath() != "/output/subs.srt" {
		t.Errorf("Expected output path '/output/subs.srt', got '%s'", builder.GetOutputPath())
	}
}

func TestSubtitleBuilder_FluentAPI(t *testing.T) {
	builder := NewSubtitleBuilder("/input/video.mp4", "/output/subs.srt").
		SetStreamIndex(1).
		SetFormat(FormatSRT).
		SetLanguage("spa").
		AddExtraArgs("-threads", "2")

	if builder.streamIndex != 1 {
		t.Error("Fluent API failed to set stream index")
	}

	if builder.format != FormatSRT {
		t.Error("Fluent API failed to set format")
	}

	if builder.language != "spa" {
		t.Error("Fluent API failed to set language")
	}
}

func TestSubtitleBuilder_AllFormats(t *testing.T) {
	formats := []SubtitleFormat{
		FormatSRT,
		FormatASS,
		FormatSSA,
		FormatVTT,
		FormatSUB,
		FormatSBV,
		FormatMOV,
	}

	for _, format := range formats {
		builder := NewSubtitleBuilder("/input/video.mp4", "/output/subs."+string(format))
		builder.SetFormat(format)

		args := builder.BuildArgs()
		argsStr := strings.Join(args, " ")

		if !strings.Contains(argsStr, "-c:s "+string(format)) {
			t.Errorf("Expected codec for format %s", format)
		}
	}
}

func TestSubtitleBuilder_ExtractSpecificStream(t *testing.T) {
	// Extract 3rd subtitle stream
	builder := NewSubtitleBuilder("/input/video.mkv", "/output/subs.srt")
	builder.SetStreamIndex(2). // 0-based, so 3rd stream
					SetFormat(FormatSRT)

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "-map 0:s:2") {
		t.Error("Expected mapping to 3rd subtitle stream")
	}
}

func TestSubtitleBuilder_ExtractByLanguage(t *testing.T) {
	builder := NewSubtitleBuilder("/input/video.mkv", "/output/french.srt")
	builder.SetLanguage("fra").SetFormat(FormatSRT)

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "-map 0:m:language:fra") {
		t.Error("Expected language-based stream selection")
	}
}

func TestSubtitleBuilder_BurnInComplex(t *testing.T) {
	// Real-world scenario: Burn subtitles with custom styling
	builder := NewSubtitleBuilder("/input/movie.mp4", "/output/movie_subbed.mp4")
	builder.BurnIntoVideo("/input/subtitles.srt").
		SetBurnInStyle("FontName=Arial,FontSize=20,PrimaryColour=&H00FFFFFF").
		AddExtraArgs("-preset", "fast", "-crf", "23")

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	// Verify key components
	if !strings.Contains(argsStr, "subtitles=") {
		t.Error("Expected subtitles filter")
	}

	if !strings.Contains(argsStr, "force_style='FontName=Arial") {
		t.Error("Expected custom styling")
	}

	if !strings.Contains(argsStr, "-c:a copy") {
		t.Error("Expected audio copy")
	}

	if !strings.Contains(argsStr, "-preset fast") {
		t.Error("Expected preset argument")
	}
}

func TestSubtitleBuilder_MultipleStreamsScenario(t *testing.T) {
	// Extract English subtitles from multi-language video
	builder := NewSubtitleBuilder("/input/multilang.mkv", "/output/english.srt")
	builder.SetLanguage("eng").
		SetFormat(FormatSRT).
		SetPriority(5)

	args := builder.BuildArgs()
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "-map 0:m:language:eng") {
		t.Error("Expected English language filter")
	}

	if !strings.Contains(argsStr, "-c:s srt") {
		t.Error("Expected SRT output format")
	}
}
