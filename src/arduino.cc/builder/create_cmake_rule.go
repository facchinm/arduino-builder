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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"arduino.cc/builder/builder_utils"
	"arduino.cc/builder/constants"
	"arduino.cc/builder/types"
	"arduino.cc/builder/utils"
)

var ADDITIONAL_FILE_VALID_EXPORT_EXTENSIONS = map[string]bool{".h": true, ".c": true, ".hpp": true, ".hh": true, ".cpp": true, ".s": true, ".a": true}

type ExportProjectCMake struct{}

func (s *ExportProjectCMake) Run(ctx *types.Context) error {
	//verbose := ctx.Verbose
	logger := ctx.GetLogger()

	// Create new cmake subFolder
	cmakeFolder := filepath.Join(ctx.BuildPath, "_cmake")
	if _, err := os.Stat(cmakeFolder); err == nil {
		os.RemoveAll(cmakeFolder)
	}
	os.Mkdir(cmakeFolder, 0777)

	libBaseFolder := filepath.Join(cmakeFolder, "lib")
	os.Mkdir(libBaseFolder, 0777)

	coreFolder := filepath.Join(cmakeFolder, "core")

	cmakeFile := filepath.Join(cmakeFolder, "CMakeLists.txt")
	//coreFolder := buildProperties[constants.BUILD_PROPERTIES_BUILD_CORE_PATH]
	//variantFolder := buildProperties[constants.BUILD_PROPERTIES_BUILD_VARIANT_PATH]

	// Copy used core + used libraries + preprocessed sketch in their folder

	// Extract CFLAGS, CPPFLAGS and LDFLAGS
	extensions := func(ext string) bool { return ADDITIONAL_FILE_VALID_EXPORT_EXTENSIONS[ext] }

	for _, library := range ctx.ImportedLibraries {
		libFolder := filepath.Join(libBaseFolder, library.Name)
		utils.CopyDir(library.Folder, libFolder, extensions)
	}

	err := utils.CopyDir(ctx.BuildProperties[constants.BUILD_PROPERTIES_BUILD_CORE_PATH], coreFolder, extensions)
	if err != nil {
		fmt.Println(err)
	}

	err = utils.CopyDir(ctx.BuildProperties[constants.BUILD_PROPERTIES_BUILD_VARIANT_PATH], filepath.Join(coreFolder, "variant"), extensions)
	if err != nil {
		fmt.Println(err)
	}

	err = utils.CopyDir(filepath.Dir(ctx.Sketch.MainFile.Name), filepath.Join(cmakeFolder, "sketch"), extensions)
	if err != nil {
		fmt.Println(err)
	}

	utils.WriteFile(filepath.Join(cmakeFolder, "sketch", filepath.Base(ctx.Sketch.MainFile.Name)+".cpp"), ctx.Sketch.MainFile.Source)

	var defines string
	//var libs string

	command, _ := builder_utils.PrepareCommandForRecipe(ctx.BuildProperties, constants.RECIPE_C_COMBINE_PATTERN, true, true, true, logger)

	for _, arg := range command.Args {
		if strings.HasPrefix(arg, "-D") {
			defines += defines + " " + arg
		}
	}

	command, _ = builder_utils.PrepareCommandForRecipe(ctx.BuildProperties, constants.RECIPE_C_PATTERN, true, true, true, logger)

	for _, arg := range command.Args {
		if strings.HasPrefix(arg, "-D") {
			defines += defines + " " + arg
		}
	}

	cmakelist := "cmake_minimum_required(VERSION 2.8.9)\n"
	cmakelist += "project (" + filepath.Base(ctx.Sketch.MainFile.Name) + ")\n"
	cmakelist += "add_definitions (" + defines + ")\n"
	cmakelist += "include_directories (core/variant core lib sketch)\n"
	cmakelist += "file (GLOB_RECURSE SOURCES ${PROJECT_SOURCE_DIR}/*.c*)\n"
	cmakelist += "file (GLOB_RECURSE SOURCES_LIBS ${PROJECT_SOURCE_DIR}/*.a)\n"
	cmakelist += "add_executable (" + filepath.Base(ctx.Sketch.MainFile.Name) + " ${SOURCES} ${SOURCES_LIBS})\n"

	utils.WriteFile(cmakeFile, cmakelist)

	for _, library := range ctx.ImportedLibraries {
		fmt.Println(library.Folder)
	}

	return nil
}
