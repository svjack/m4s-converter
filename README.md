#### 构建视频数据集

```python
import os
import shutil
import json

def copy_and_rename_files(source_root, dest_images, dest_videos):
    # 确保目标文件夹存在
    os.makedirs(dest_images, exist_ok=True)
    os.makedirs(dest_videos, exist_ok=True)

    # 遍历源路径下的所有文件夹
    for folder_name in os.listdir(source_root):
        folder_path = os.path.join(source_root, folder_name)
        
        if not os.path.isdir(folder_path):
            continue  # 跳过非文件夹

        # 查找 videoinfo.json
        json_path = os.path.join(folder_path, "videoinfo.json")
        if not os.path.exists(json_path):
            print(f"⚠️ No videoinfo.json found in {folder_path}")
            continue

        # 解析 JSON 获取 title
        try:
            with open(json_path, "r", encoding="utf-8") as f:
                video_info = json.load(f)
                title = video_info.get("title", "").strip()
                if not title:
                    print(f"⚠️ Empty title in {json_path}")
                    continue
        except Exception as e:
            print(f"❌ Failed to parse {json_path}: {e}")
            continue

        # 查找 image.jpg 和 .mp4 文件
        image_file = None
        mp4_file = None

        for file in os.listdir(folder_path):
            file_lower = file.lower()
            if file_lower == "image.jpg":
                image_file = os.path.join(folder_path, file)
            elif file_lower.endswith(".mp4"):
                mp4_file = os.path.join(folder_path, file)

        # 处理非法文件名字符（替换为下划线）
        safe_title = "".join(c if c.isalnum() or c in (" ", "-", "_") else "_" for c in title)

        # 拷贝并重命名 image.jpg
        if image_file:
            dest_image = os.path.join(dest_images, f"{safe_title}.jpg")
            shutil.copy2(image_file, dest_image)
            print(f"✅ Copied image: {image_file} → {dest_image}")
        else:
            print(f"⚠️ No image.jpg found in {folder_path}")

        # 拷贝并重命名 .mp4
        if mp4_file:
            dest_video = os.path.join(dest_videos, f"{safe_title}.mp4")
            shutil.copy2(mp4_file, dest_video)
            print(f"✅ Copied video: {mp4_file} → {dest_video}")
        else:
            print(f"⚠️ No .mp4 file found in {folder_path}")

if __name__ == "__main__":
    # 源文件夹路径
    source_path = r"C:\Users\DELL\Videos\bilibili\output\原神-sketch"

    # 目标文件夹路径（存放图片和视频）
    dest_images = r"C:\Users\DELL\Videos\bilibili\output\images"
    dest_videos = r"C:\Users\DELL\Videos\bilibili\output\videos"

    copy_and_rename_files(source_path, dest_images, dest_videos)
    print("🎉 文件拷贝和重命名完成！")
```

#### 视频抽象功能
```python
from dataclasses import dataclass
from pathlib import Path
from typing import Optional
import cv2
import numpy as np
import logging

logger = logging.getLogger(__name__)

@dataclass
class Frame:
    number: int
    path: Path
    timestamp: float
    score: float

def create_abstract_video(
    input_video_path: Path,
    output_video_path: Path,
    frames_per_minute: int = 10,
    duration: Optional[float] = None,
    max_frames: Optional[int] = None,
    frame_difference_threshold: float = 1.0,
    target_duration: Optional[float] = 4
) -> None:
    """
    Create an abstract video by extracting and saving keyframes from the input video,
    maintaining the original temporal order of the frames and optionally adjusting to target duration.
    
    Args:
        input_video_path: Path to the input video file
        output_video_path: Path where the abstract video will be saved
        frames_per_minute: Target number of frames per minute in output
        duration: Maximum duration (in seconds) to process from input video
        max_frames: Maximum number of frames to include in output
        frame_difference_threshold: Minimum difference score to consider a frame as keyframe
        target_duration: Desired duration of output video in seconds (frames will be evenly distributed)
    """
    def _calculate_frame_difference(frame1: np.ndarray, frame2: np.ndarray) -> float:
        """Calculate the difference between two frames using absolute difference."""
        if frame1 is None or frame2 is None:
            return 0.0
        
        gray1 = cv2.cvtColor(frame1, cv2.COLOR_BGR2GRAY)
        gray2 = cv2.cvtColor(frame2, cv2.COLOR_BGR2GRAY)
        diff = cv2.absdiff(gray1, gray2)
        return float(np.mean(diff))

    # Open input video
    cap = cv2.VideoCapture(str(input_video_path))
    if not cap.isOpened():
        raise ValueError(f"Could not open video file: {input_video_path}")
    
    fps = cap.get(cv2.CAP_PROP_FPS)
    total_frames = int(cap.get(cv2.CAP_PROP_FRAME_COUNT))
    video_duration = total_frames / fps
    
    if duration:
        video_duration = min(duration, video_duration)
        total_frames = int(min(total_frames, duration * fps))
    
    # Calculate target number of frames
    target_frames = max(1, min(
        int((video_duration / 60) * frames_per_minute),
        total_frames,
        max_frames if max_frames is not None else float('inf')
    ))
    
    # Calculate adaptive sampling interval
    sample_interval = max(1, total_frames // (target_frames * 2))
    
    # Extract keyframe candidates while maintaining order
    frame_candidates = []
    prev_frame = None
    frame_count = 0
    
    while frame_count < total_frames:
        ret, frame = cap.read()
        if not ret:
            break
            
        if frame_count % sample_interval == 0:
            score = _calculate_frame_difference(frame, prev_frame)
            if score > frame_difference_threshold:
                timestamp = frame_count / fps
                frame_candidates.append((frame_count, frame, score, timestamp))
            prev_frame = frame.copy()
            
        frame_count += 1
        
    cap.release()
    
    # Sort candidates by frame number to maintain temporal order
    frame_candidates.sort(key=lambda x: x[0])
    
    # Select frames evenly spaced while preserving order
    if len(frame_candidates) > target_frames:
        step = len(frame_candidates) / target_frames
        indices = [int(i * step) for i in range(target_frames)]
        selected_frames = [frame_candidates[i] for i in indices]
    else:
        selected_frames = frame_candidates

    # Adjust frame display durations if target_duration is specified
    if target_duration is not None and target_duration > 0:
        # Calculate how many times each frame should be repeated
        total_frames_needed = int(target_duration * fps)
        frames_count = len(selected_frames)
        
        if frames_count == 0:
            raise ValueError("No keyframes were selected for the output video")
            
        # Calculate repeat counts for each frame to evenly distribute
        base_repeats = total_frames_needed // frames_count
        remainder = total_frames_needed % frames_count
        
        frame_repeats = [base_repeats + 1 if i < remainder else base_repeats 
                         for i in range(frames_count)]
    else:
        # Default: each frame appears once
        frame_repeats = [1] * len(selected_frames)

    # Create output video writer
    if not selected_frames:
        raise ValueError("No keyframes were selected for the output video")
    
    frame_height, frame_width = selected_frames[0][1].shape[:2]
    fourcc = cv2.VideoWriter_fourcc(*'mp4v')
    
    # Use original fps if no target duration, otherwise calculate new fps to match duration
    output_fps = fps if target_duration is None else (sum(frame_repeats) / target_duration)
    out = cv2.VideoWriter(str(output_video_path), fourcc, output_fps, (frame_width, frame_height))
    
    # Write selected frames to output video with proper repeats
    for (frame_num, frame, score, timestamp), repeats in zip(selected_frames, frame_repeats):
        for _ in range(repeats):
            out.write(frame)
    
    out.release()
    
    actual_duration = sum(frame_repeats) / output_fps
    logger.info(f"Created abstract video with {len(selected_frames)} keyframes "
                f"at {output_video_path}, duration: {actual_duration:.2f}s")

# Example usage:
# create_abstract_video(Path("input.mp4"), Path("output.mp4"), 
#                      frames_per_minute=15, target_duration=30)  # 30 second output

create_abstract_video(
    r"C:\Users\DELL\Downloads\Genshin_StarRail_Longshu_Sketch_Tail_Videos_Reversed\reversed_卡维兴奋了___ - Made with Clipchamp.mp4",
    "reversed_卡维兴奋了___ - Made with Clipchamp.mp4",
    frames_per_minute = 60
)

create_abstract_video(
    r"C:\Users\DELL\Downloads\Genshin_StarRail_Longshu_Sketch_Tail_Videos_Reversed\reversed_海哥_卡维默契了一把 - Made with Clipchamp.mp4",
    "reversed_海哥_卡维默契了一把 - Made with Clipchamp.mp4",
    frames_per_minute = 60,
    target_duration = 2
)
```

## 为什么开发此程序？
bilibili下架了很多视频，之前收藏和缓存的视频均无法播放

![image](https://github.com/mzky/m4s-converter/assets/13345233/ea8bc799-e47d-40ca-bde4-c47193f0e453)

- 喜欢的视频赶紧缓存起来，使用本程序将bilibili缓存的m4s转成mp4，方便随时播放。

- 因bilibili使用的是GPAC处理视频，本工具从v1.5.0开始默认使用GPAC的MP4Box进行音视频合成（此版开始不支持32位系统），能够避免FFMpeg合成视频后音画不同步问题，详见：https://github.com/mzky/m4s-converter/issues/11


### 下载后双击执行或通过命令行执行，需要可执行权限
- https://github.com/mzky/m4s-converter/releases/latest


### Android手机端合并文件方法 
- 详见：[拷贝文件与合成方法](https://github.com/mzky/m4s-converter/issues/9)


### 除window和linux外，其它环境的依赖工具安装
- 详见：[依赖工具安装](https://github.com/mzky/m4s-converter/wiki/%E4%BE%9D%E8%B5%96%E5%B7%A5%E5%85%B7%E5%AE%89%E8%A3%85)


### 命令行参数
```
# 指定FFMpeg路径: ./m4s-converter-linux_amd64 -f /var/FFMpeg/ffmpeg 或 ./m4s-converter-amd64 -f select
# 指定MP4Box路径: ./m4s-converter-amd64.exe -g "D:\GPAC\mp4box.exe" 或 ./m4s-converter-amd64 -g select
 Flags: 
    -h --help         查看帮助信息
    -v --version      查看版本信息
    -a --assoff       关闭自动生成弹幕功能，默认不关闭
    -s --skip         跳过合成同名视频(优先级高于overlay)，默认不跳过，但会跳过[完全相同]的文件
    -o --overlay      合成文件时是否覆盖同名视频，默认不覆盖并重命名新文件
    -c --cachepath    自定义视频缓存路径，默认使用bilibili的默认缓存路径
    -g --gpacpath     自定义GPAC的mp4box文件路径,值为select时弹出选择对话框
    -f --ffmpegpath   自定义FFMpeg文件路径,值为select时弹出选择对话框
```


### 验证合成：
```
2023-12-05_16:02:46 [INFO ] 已合成视频文件:中国-美景极致享受-笨蹦崩.mp4
2023-12-05_16:02:46 [INFO ] ==========================================
2023-12-05_16:02:46 [INFO ] 合成的文件:
C:\Users\mzky\Videos\bilibili\output\【获奖学生动画】The Little Poet 小诗人｜CALARTS 2023\【获奖学生动画】The Little Poet 小诗人｜CALARTS 2023-toh糖.mp4
C:\Users\mzky\Videos\bilibili\output\【电影历史_专题片】《影响》致敬中国电影40年【全集】\40年光影记忆-开飞机的巡查司.mp4
C:\Users\mzky\Videos\bilibili\output\“我不是个好导演”，听田壮壮讲述“我和电影的关系”\“我不是个好导演”，听田壮壮讲述“我和电影的关系”-Tatler的朋友们.mp4
C:\Users\mzky\Videos\bilibili\output\【4K8K-世界各地的美景】\中国-美景极致享受-笨蹦崩.mp4
2023-12-05_16:02:46 [INFO ] 已完成本次任务，耗时:5秒
2023-12-05_16:02:46 [INFO ] ==========================================
按回车键退出...
```

- 合成 1.46GB 文件，耗时: 5 秒
- 合成 11.7GB 文件，耗时:38 秒

以上为固态硬盘测试结果, 仅供参考

##
#### 弹幕xml转换为ass使用了此项目
- https://github.com/kafuumi/converter


#### 视频编码使用的工具
- https://gpac.io
- https://ffmpeg.org
- 本程序不会对下载的音视频转码，仅通过上面两个工具进行音视频轨合成


#### 非缓存方式下载，推荐使用其它工具
- https://github.com/nICEnnnnnnnLee/BilibiliDown
- https://github.com/leiurayer/downkyi


## 提缺陷和建议

知乎不常上，缺陷或建议提交 [issues](https://github.com/mzky/m4s-converter/issues/new/choose) , 最好带上异常视频的URL地址
