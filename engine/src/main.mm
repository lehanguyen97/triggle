#define GLFW_INCLUDE_NONE
#define GLFW_EXPOSE_NATIVE_COCOA
#include <GLFW/glfw3.h>
#include <GLFW/glfw3native.h>

#import <Metal/Metal.h>
#import <QuartzCore/CAMetalLayer.h>

#include <stdio.h>

#define SOKOL_IMPL
#include "engine.hpp"

CAMetalLayer *layer;
id<MTLDevice> device;

sg_swapchain get_swapchain(void) {
  id<CAMetalDrawable> next_drawable = [layer nextDrawable];
  const CGSize fb_size = layer.drawableSize;
  return (sg_swapchain){.width = (int)fb_size.width,
      .height = (int)fb_size.height,
      .sample_count = 1,
      .color_format = SG_PIXELFORMAT_BGRA8,
      .depth_format = SG_PIXELFORMAT_NONE,
      .metal = {
          .current_drawable = (__bridge const void *)next_drawable,
      }};
}

sg_environment get_environment(void) {
  return (sg_environment){.defaults =
                              {
                                  .color_format = SG_PIXELFORMAT_BGRA8,
                                  .depth_format = SG_PIXELFORMAT_NONE,
                                  .sample_count = 1,
                              },
      .metal = {
          .device = (__bridge const void *)device,
      }};
}

int main() {
  printf("Starting Triggle...\n");

  // Create Metal device
  device = MTLCreateSystemDefaultDevice();
  layer = [CAMetalLayer layer];
  layer.device = device;
  layer.opaque = YES;

  if (!glfwInit()) {
    return -1;
  }

  glfwWindowHint(GLFW_CLIENT_API, GLFW_NO_API);
  glfwWindowHint(GLFW_RESIZABLE, GLFW_TRUE);

  GLFWwindow *window = glfwCreateWindow(1280, 720, "Triggle", NULL, NULL);
  if (!window) {
    glfwTerminate();
    return -1;
  }

  // Get the Cocoa window
  NSWindow *nsWindow = glfwGetCocoaWindow(window);

  // Attach Metal layer to window
  nsWindow.contentView.layer = layer;
  nsWindow.contentView.wantsLayer = YES;

  Engine engine;
  engine.init();

  // render loop
  while (!glfwWindowShouldClose(window)) {
    glfwPollEvents();

    // Update layer size if window was resized
    int width, height;
    glfwGetFramebufferSize(window, &width, &height);
    layer.drawableSize = CGSizeMake(width, height);

    engine.frame();
  }

  printf("Cleaning up...\n");

  // Cleanup
  engine.shutdown();
  glfwTerminate();

  printf("Triggle finished successfully!\n");
  return 0;
}