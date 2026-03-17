####
# 'SpaceDrone' Deployment:
#
# F Prime deployment for the HADES Autonomous Space Drone.
# Implements ESS paper architecture with 6 flight software components:
#   MissionControl, GNC, PowerManagement, Communications,
#   ObjectDetection, HealthMonitor
#
# Reference: Nelson, G. "Autonomous Space UAVs: Embedded Systems
# Architecture for Intergalactic Drones with NASA F Prime and
# Cloud-Native Ground Control Systems." Oxford, 2026.
####

cmake_policy(SET CMP0048 NEW)
cmake_minimum_required(VERSION 3.26)
project(SpaceDrone VERSION 1.0.0 LANGUAGES C CXX)

add_compile_options(
    $<$<COMPILE_LANGUAGE:CXX>:-Wold-style-cast>
    -Wall
    -Wextra
    -Werror
    -Wno-unused-parameter
    -pedantic
)

# Find the F Prime package from NASA's framework
set(FPRIME_INCLUDE_FRAMEWORK_CODE ON)
find_package(FPrime REQUIRED PATHS "${CMAKE_CURRENT_LIST_DIR}/../fprime-framework")

# Add SpaceDrone subsystem components (Types first — other components depend on it)
add_fprime_subdirectory("${CMAKE_CURRENT_LIST_DIR}/SpaceDrone/Types/")
add_fprime_subdirectory("${CMAKE_CURRENT_LIST_DIR}/SpaceDrone/MissionControl/")
add_fprime_subdirectory("${CMAKE_CURRENT_LIST_DIR}/SpaceDrone/GNC/")
add_fprime_subdirectory("${CMAKE_CURRENT_LIST_DIR}/SpaceDrone/PowerManagement/")
add_fprime_subdirectory("${CMAKE_CURRENT_LIST_DIR}/SpaceDrone/Communications/")
add_fprime_subdirectory("${CMAKE_CURRENT_LIST_DIR}/SpaceDrone/ObjectDetection/")
add_fprime_subdirectory("${CMAKE_CURRENT_LIST_DIR}/SpaceDrone/HealthMonitor/")
