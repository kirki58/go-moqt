package h264

import (
	"bufio"
	"go-moq/pkg/model"
	"go-moq/pkg/session"
	"io"
	"log"
	"os/exec"

	"github.com/pion/webrtc/v3/pkg/media/h264reader"
)

type H264Encoder struct{
	dispatcher *session.Dispatcher
	ftn        *model.MoqtFullTrackName
}

func (en *H264Encoder) Encode(source_path string){
	// The FFmpeg command arguments
	args := []string{
		"-re",                             // Read input in real-time
		"-f", "lavfi", "-i", "testsrc=duration=120:size=854x480:rate=30",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-pix_fmt", "yuv420p",
		"-g", "30",                        // GOP size (1 IDR per second)
		"-keyint_min", "30",               // Force fixed GOP
		"-sc_threshold", "0",              // No accidental I-frames
		"-x264-params", "repeat-headers=1", // SPS/PPS before every IDR
		"-bsf:v", "h264_mp4toannexb",      // Ensure Annex B start codes
		"-f", "h264",                      // Raw H.264 elementary stream
		"-",                               // Output to Stdout
	}

	cmd := exec.Command("ffmpeg", args...)
	
	// Connect to the Stdout of the process
	stdout, err := cmd.StdoutPipe()
	if err != nil{
		log.Printf("[Encoder ERROR] Could not create stdout pipe from ffmpeg process / %v\n", err)
		return
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		log.Printf("[Encoder ERROR] Error starting ffmpeg process / %v\n", err)
		return
	}

	log.Printf("[Encoder INFO] Started ffmpeg process with PID %d\n", cmd.Process.Pid)

	reader := bufio.NewReader(stdout)
	h264R, err := h264reader.NewReader(reader)
	if err != nil {
		log.Printf("[Encoder ERROR] Failed to create H264 reader / %v\n", err)
		return
	}

	log.Println("[Encoder] Starting Annex B NALU Extraction")

	latestGroup := uint64(0)
	latestObj := uint64(0)

	for {
		nal, err := h264R.NextNAL()
		if err != nil{
			if err == io.EOF{
				// End of track
				break
			}
			log.Printf("[Encoder ERROR] Error reading NALU: %v\n", err)
			continue
		}

		if len(nal.Data) == 0{
			continue
		}

		// First byte of the NALU is the header (IPR SPS/PPS etc.)
		headerByte := nal.Data[0]
		// Lower 5 bits of the header indicate it's type (AND with 00011111 to get the lower 5 bits)
		nalUnitType := headerByte & 0x1F

		var payload []byte
		switch nalUnitType {
		case 1:
			// Classic P frame
			obj := &model.MoqtObject{
				Location: model.MoqtLocation{GroupId: latestGroup, ObjectId: latestObj},
				SubgroupID: 0,
				FullTrackName: *en.ftn,
				PublisherPriority: 128,
				ObjectForwardingPreference: model.Subgroup,
				ObjectStatus: model.Normal,
				Payload: nal.Data,
			}
			en.dispatcher.Dispatch(obj)
			latestObj++

		case 7:
			// SPS Unit, encoder configuration metadata
			// Acts as group start
			// MUST read 1 PPS and 1 IDR units in that order to get a full group 
			// It is guaranteed to receive SPS, PPS, and IDR units in that order per the ffmpeg encoder
			payload = append(payload, nal.Data...)

			latestGroup++
			latestObj = 0

		case 8:
			// PPS Unit, picture metadata
			// Received after SPS
			// It should be inside the group start object
			payload = append(payload, nal.Data...)
		
		case 5:
			// IDR Unit, Contains the keyframe
			// Received after PPS
			// It should be the last unit inside the group start object
			payload = append(payload, nal.Data...)
			obj := &model.MoqtObject{
				Location: model.MoqtLocation{GroupId: latestGroup, ObjectId: latestObj},
				SubgroupID: 0,
				FullTrackName: *en.ftn,
				PublisherPriority: 128,
				ObjectForwardingPreference: model.Subgroup,
				ObjectStatus: model.Normal,
				Payload: payload,
			}
			en.dispatcher.Dispatch(obj)
			payload = payload[:0] // empty the paylaod buffer for the next group start (keep the capacity) 
		}
	}
	en.dispatcher.CloseAll()


	cmd.Wait()
}