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
 */

package builder

import (
	"regexp"
	"strings"

	"arduino.cc/builder/types"
)

type FindAndApplySketchPreprocessorDirectives struct{}

func (s *FindAndApplySketchPreprocessorDirectives) Run(ctx *types.Context) error {

	sketch := ctx.Sketch.MainFile.Source
	rewritesProperties := extractPreprocessorDirectives(sketch)

	ctx.CustomBuildProperties = append(ctx.CustomBuildProperties, rewritesProperties...)

	return nil
}

func extractPreprocessorDirectives(sketch string) []string {
	/*
		In a very CGO-like fashion, search for strings matching
		// #arduino {directive}
		declared in the main sketch before the first line of code
		use the directive to populate an extra preference map
	*/

	var properties []string

	firstCodeChar := getFirstNonCommentNonBlankCharacter(sketch)

	r, _ := regexp.Compile("(?m)^//\\s*#arduino\\s*.*=.*$")
	results := r.FindAllString(sketch, -1)
	resIdx := r.FindAllStringIndex(sketch, -1)

	for i, result := range results {
		result = strings.Replace(result, "//", "", 1)
		result = strings.Replace(result, "#arduino", "", 1)
		result = strings.TrimSpace(result)
		if resIdx[i][0] < firstCodeChar {
			properties = append(properties, result)
		}
	}
	return properties
}

func getFirstNonCommentNonBlankCharacter(text string) int {

	lines := strings.Split(text, "\n")

	characters := 0

	cppStyleComment, _ := regexp.Compile("(?m)^(\\s*)//")
	cStyleCommentOneline, _ := regexp.Compile("(?m)^(\\s*)/\\*.*\\*/(\\s*)$")
	cStyleStartComment, _ := regexp.Compile("(?m)^(\\s*)/\\*")
	cStyleEndComment, _ := regexp.Compile("(?m).*\\*/(\\s*)$")
	emptyLine, _ := regexp.Compile("(?m)^\\s*$")

	multilineComment := false

	for _, line := range lines {
		if emptyLine.MatchString(line) {
			characters += len(line)
			continue
		}
		if cppStyleComment.MatchString(line) || cStyleCommentOneline.MatchString(line) {
			characters += len(line)
			continue
		}
		if cStyleStartComment.MatchString(line) {
			multilineComment = true
			characters += len(line)
			continue
		}
		if cStyleEndComment.MatchString(line) && multilineComment {
			multilineComment = false
			characters += len(line)
			continue
		}
		if multilineComment == true {
			characters += len(line)
			continue
		}
		break
	}
	return characters
}
