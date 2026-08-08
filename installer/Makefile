# Makefile - 使用 MinGW-w64 构建 setup.exe

CXX      := g++
WINDRES  := windres
CXXFLAGS := -std=c++17 -O2 -DUNICODE -D_UNICODE -mwindows -Wall -D_WIN32_WINNT=0x0A00 -DWINVER=0x0A00
LDFLAGS  := -mwindows -static -static-libgcc -static-libstdc++ -lgdiplus -lgdi32 -lcomctl32 -luuid -lshlwapi -lole32 -lshell32 -ladvapi32 -ldwmapi

SRC_DIR  := src
RES_DIR  := res
OBJ_DIR  := obj

# 源文件
SOURCES := $(wildcard $(SRC_DIR)/*.cpp)
OBJECTS := $(patsubst $(SRC_DIR)/%.cpp,$(OBJ_DIR)/%.o,$(SOURCES))

# 资源编译产物
RESOURCE_OBJ := $(OBJ_DIR)/installer_res.o

TARGET := setup.exe

.PHONY: all clean

all: $(TARGET)

$(TARGET): $(OBJECTS) $(RESOURCE_OBJ)
	$(CXX) $(OBJECTS) $(RESOURCE_OBJ) -o $@ $(LDFLAGS)

$(OBJ_DIR)/%.o: $(SRC_DIR)/%.cpp | $(OBJ_DIR)
	$(CXX) $(CXXFLAGS) -c $< -o $@

$(RESOURCE_OBJ): $(RES_DIR)/installer.rc $(wildcard $(RES_DIR)/*) | $(OBJ_DIR)
	$(WINDRES) -I$(SRC_DIR) -i $< -o $@

$(OBJ_DIR):
	mkdir -p $(OBJ_DIR)

clean:
	rm -rf $(OBJ_DIR) $(TARGET)
