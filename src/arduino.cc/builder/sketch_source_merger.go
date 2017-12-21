/*
 * This file is part of Arduino Builder.
 *
 * Arduino Builder is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA  02110-1301  USA
 *
 * As a special exception, you may use this file as part of a free software
 * library without restriction.  Specifically, if other files instantiate
 * templates or use macros or inline functions from this file, or you compile
 * this file and link it with other files to produce an executable, this
 * file does not by itself cause the resulting executable to be covered by
 * the GNU General Public License.  This exception does not however
 * invalidate any other reasons why the executable file might be covered by
 * the GNU General Public License.
 *
 * Copyright 2015 Arduino LLC (http://www.arduino.cc/)
 * Copyright 2015 Matthijs Kooijman
 */

package builder

import (
	"regexp"
	"strconv"
	"strings"

	"arduino.cc/builder/types"
	"arduino.cc/builder/utils"
)

type AddArduinoAndRestoreIncludes struct{}

func (s *AddArduinoAndRestoreIncludes) Run(ctx *types.Context) error {
	includeSection := ""
	if !sketchIncludesArduinoH(&ctx.Sketch.MainFile) {
		includeSection += "#line 1 " + utils.QuoteCppString(ctx.Sketch.MainFile.Name) + "\n"
		includeSection += "#include <Arduino.h>\n"
	}
	for _, include := range ctx.Includes {
		includeSection += include.LineMarker + "\n"
		includeSection += include.Content + "\n"
	}
	ctx.Source = includeSection + ctx.Source
	return nil
}

type SaveIncludes struct{}

func (s *SaveIncludes) Run(ctx *types.Context) error {
	// grab includes from all source files
	// save them in a struct
	sketch := ctx.Sketch
	var includes []types.Include
	includes = append(includes, parseSketchForIncludes(&sketch.MainFile)...)
	for _, file := range sketch.OtherSketchFiles {
		includes = append(includes, parseSketchForIncludes(&file)...)
	}
	ctx.Includes = includes
	return nil
}

func parseSketchForIncludes(sketch *types.SketchFile) []types.Include {
	var includes []types.Include
	re := regexp.MustCompile("(?m)^\\s*#\\s*include\\s*[<\"](.*?)[>\"]")
	matches := re.FindAllString(sketch.Source, -1)
	splittedByLines := strings.Split(sketch.Source, "\n")
	for _, match := range matches {
		for i, line := range splittedByLines {
			if strings.Contains(line, strings.TrimSpace(match)) {
				includes = append(includes, types.Include{Content: line, LineMarker: "# " + strconv.Itoa(i+1) + " " + utils.QuoteCppString(sketch.Name)})
				break
			}
		}
	}
	return includes
}

type SketchSourceMerger struct{}

func (s *SketchSourceMerger) Run(ctx *types.Context) error {
	sketch := ctx.Sketch

	lineOffset := 0
	includeSection := ""
	includeSection += "#line 1 " + utils.QuoteCppString(sketch.MainFile.Name) + "\n"
	lineOffset++
	if !sketchIncludesArduinoH(&sketch.MainFile) {
		includeSection += "#include <Arduino.h>\n"
		lineOffset++
	}
	ctx.IncludeSection = includeSection

	source := includeSection
	source += addSourceWrappedWithLineDirective(&sketch.MainFile)
	lineOffset += 1
	for _, file := range sketch.OtherSketchFiles {
		source += addSourceWrappedWithLineDirective(&file)
	}

	ctx.LineOffset = lineOffset
	ctx.Source = source

	return nil
}

func sketchIncludesArduinoH(sketch *types.SketchFile) bool {
	if matched, err := regexp.MatchString("(?m)^\\s*#\\s*include\\s*[<\"]Arduino\\.h[>\"]", sketch.Source); err != nil {
		panic(err)
	} else {
		return matched
	}
}

func addSourceWrappedWithLineDirective(sketch *types.SketchFile) string {
	source := "#line 1 " + utils.QuoteCppString(sketch.Name) + "\n"
	source += sketch.Source
	source += "\n"

	return source
}
