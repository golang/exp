#include <stdint.h>

void startDriver();
void stopDriver();
void makeCurrentContext(uintptr_t ctx);
uintptr_t doNewWindow(int width, int height);
uintptr_t doShowWindow(uintptr_t id);
void doCloseWindow(uintptr_t id);
uint64_t threadID();
