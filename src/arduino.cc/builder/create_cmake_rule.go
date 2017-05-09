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
	"arduino.cc/builder/i18n"
	"arduino.cc/builder/types"
	"arduino.cc/builder/utils"
)

var ADDITIONAL_FILE_VALID_EXPORT_EXTENSIONS = map[string]bool{".h": true, ".c": true, ".hpp": true, ".hh": true, ".cpp": true, ".s": true, ".a": true}
var DOTHEXTENSION = map[string]bool{".h": true, ".hh": true, ".hpp": true}
var DOTAEXTENSION = map[string]bool{".a": true}

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

	buildBaseFolder := filepath.Join(cmakeFolder, "build")
	os.Mkdir(buildBaseFolder, 0777)

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

	err = utils.CopyDir(ctx.SketchBuildPath, filepath.Join(cmakeFolder, "sketch"), extensions)
	if err != nil {
		fmt.Println(err)
	}

	//utils.WriteFile(filepath.Join(cmakeFolder, "sketch", filepath.Base(ctx.Sketch.MainFile.Name)+".cpp"), ctx.Sketch.MainFile.Source)

	var defines []string
	var linkerflags []string
	var libs []string
	var linkDirectories []string

	extractCompileFlags(ctx, constants.RECIPE_C_COMBINE_PATTERN, &defines, &libs, &linkerflags, &linkDirectories, logger)
	extractCompileFlags(ctx, constants.RECIPE_C_PATTERN, &defines, &libs, &linkerflags, &linkDirectories, logger)
	extractCompileFlags(ctx, constants.RECIPE_CPP_PATTERN, &defines, &libs, &linkerflags, &linkDirectories, logger)

	var headerFiles []string
	isHeader := func(ext string) bool { return DOTHEXTENSION[ext] }
	utils.FindFilesInFolder(&headerFiles, cmakeFolder, isHeader, true)
	foldersContainingDotH := findUniqueFoldersRelative(headerFiles, cmakeFolder)

	var staticLibsFiles []string
	isStaticLib := func(ext string) bool { return DOTAEXTENSION[ext] }
	utils.FindFilesInFolder(&staticLibsFiles, cmakeFolder, isStaticLib, true)
	//foldersContainingDotA := findUniqueFoldersRelative(staticLibsFiles, cmakeFolder)

	fmt.Println(libs)
	for i, _ := range libs {
		libs[i] = strings.TrimPrefix(libs[i], "-l")
	}

	cmakelist := "cmake_minimum_required(VERSION 2.8.9)\n"
	cmakelist += "INCLUDE(FindPkgConfig)\n"
	cmakelist += "project (" + filepath.Base(ctx.Sketch.MainFile.Name) + " C CXX)\n"
	cmakelist += "add_definitions (" + strings.Join(defines, " ") + " " + strings.Join(linkerflags, " ") + ")\n"
	cmakelist += "include_directories (" + foldersContainingDotH + ")\n"

	var relLinkDirectories []string
	for _, dir := range linkDirectories {
		relLinkDirectories = append(relLinkDirectories, strings.TrimPrefix(dir, cmakeFolder))
	}
	for _, lib := range libs {
		//cmakelist += "add_library (" + lib + " SHARED IMPORTED)\n"
		cmakelist += "pkg_search_module (" + strings.ToUpper(lib) + "REQUIRED " + lib + ")\n"
		linkDirectories = append(linkDirectories, "${"+strings.ToUpper(lib)+"_LIBRARY_DIRS}")
		//cmakelist += "set_property(TARGET " + lib + " PROPERTY IMPORTED_LOCATION " + location + " )\n"
	}
	cmakelist += "link_directories (" + strings.Join(linkDirectories, " ") + ")\n"
	for _, staticLibsFile := range staticLibsFiles {
		lib := filepath.Base(staticLibsFile)
		lib = strings.TrimPrefix(lib, "lib")
		lib = strings.TrimSuffix(lib, ".a")
		if !utils.SliceContains(libs, lib) {
			libs = append(libs, lib)
			cmakelist += "add_library (" + lib + " STATIC IMPORTED)\n"
			location := strings.TrimPrefix(staticLibsFile, cmakeFolder)
			cmakelist += "set_property(TARGET " + lib + " PROPERTY IMPORTED_LOCATION " + "${PROJECT_SOURCE_DIR}" + location + " )\n"
		}
	}
	cmakelist += "file (GLOB_RECURSE SOURCES core/*.c* lib/*.c* sketch/*.c*)\n"
	cmakelist += "add_executable (" + filepath.Base(ctx.Sketch.MainFile.Name) + " ${SOURCES} ${SOURCES_LIBS})\n"
	cmakelist += "target_link_libraries( " + filepath.Base(ctx.Sketch.MainFile.Name) + " " + strings.Join(libs, " ") + ")\n"

	utils.WriteFile(cmakeFile, cmakelist)

	/*
		for _, library := range ctx.ImportedLibraries {
			fmt.Println(library.Folder)
		}
	*/
	return nil
}

func extractCompileFlags(ctx *types.Context, receipe string, defines, libs, linkerflags, linkDirectories *[]string, logger i18n.Logger) {
	command, _ := builder_utils.PrepareCommandForRecipe(ctx.BuildProperties, receipe, true, true, true, logger)

	for _, arg := range command.Args {
		if strings.HasPrefix(arg, "-D") {
			*defines = appendIfUnique(*defines, arg)
		} else {
			if strings.HasPrefix(arg, "-l") {
				*libs = appendIfUnique(*libs, arg)
			} else {
				if strings.HasPrefix(arg, "-L") {
					*linkDirectories = appendIfUnique(*linkDirectories, strings.TrimPrefix(arg, "-L"))
				} else {
					if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "-I") {
						// HACK : from linkerflags remove MMD
						if !strings.HasPrefix(arg, "-MMD") {
							*linkerflags = appendIfUnique(*linkerflags, arg)
						}
					}
				}
			}
		}
	}
}

func findUniqueFoldersRelative(slice []string, base string) string {
	var out []string
	for _, element := range slice {
		path := filepath.Dir(element)
		path = strings.TrimPrefix(path, base+"/")
		if !utils.SliceContains(out, path) {
			out = append(out, path)
		}
	}
	return strings.Join(out, " ")
}

func appendIfUnique(slice []string, element string) []string {
	if !utils.SliceContains(slice, element) {
		slice = append(slice, element)
	}
	return slice
}
