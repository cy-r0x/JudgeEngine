# Makefile for testlib built-in checkers
# Run: make
# Binaries are placed in ../checker/

CXX = g++
CXXFLAGS = -O2 -std=c++17 -pipe -s

TARGET_DIR = ../checker
CHECKERS = default_checker int_checker float_checker unordered_checker strict_checker

all: $(CHECKERS)

default_checker: default_checker.cpp testlib.h
	$(CXX) $(CXXFLAGS) $< -o $(TARGET_DIR)/$@

int_checker: int_checker.cpp testlib.h
	$(CXX) $(CXXFLAGS) $< -o $(TARGET_DIR)/$@

float_checker: float_checker.cpp testlib.h
	$(CXX) $(CXXFLAGS) $< -o $(TARGET_DIR)/$@

unordered_checker: unordered_checker.cpp testlib.h
	$(CXX) $(CXXFLAGS) $< -o $(TARGET_DIR)/$@

strict_checker: strict_checker.cpp testlib.h
	$(CXX) $(CXXFLAGS) $< -o $(TARGET_DIR)/$@

clean:
	rm -f $(TARGET_DIR)/default_checker $(TARGET_DIR)/int_checker $(TARGET_DIR)/float_checker $(TARGET_DIR)/unordered_checker $(TARGET_DIR)/strict_checker

.PHONY: all clean
