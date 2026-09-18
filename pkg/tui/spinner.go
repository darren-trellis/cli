/*
Copyright © 2023 Doppler <support@doppler.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package tui

import "time"

// spinnerTickMsg advances the loading spinner.
type spinnerTickMsg struct{}

var spinnerFrames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

const spinnerInterval = 100 * time.Millisecond

// spinnerState replaces bubbles/spinner: a frame index plus a tick command.
type spinnerState struct {
	frame int
}

func (s spinnerState) View() string {
	if len(spinnerFrames) == 0 {
		return ""
	}
	return spinnerFrames[s.frame%len(spinnerFrames)]
}

func (s *spinnerState) advance() {
	s.frame = (s.frame + 1) % len(spinnerFrames)
}

// Tick schedules the next frame.
func (s spinnerState) Tick() Msg {
	time.Sleep(spinnerInterval)
	return spinnerTickMsg{}
}
