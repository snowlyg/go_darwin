<template>
  <div class="app-container">
    <div class="filter-container">
      <div class="box-card">
        <video
          id="my-video"
          class="video-js vjs-default-skin"
          controls
          preload="auto"
          width="800px"
          muted
        >
          <source id="video-source" src="http://127.0.0.1:7002/godarwin/6fd22224e8672ef7e07f32112ec59e74.m3u8" type="application/x-mpegURL">
        </video>
      </div>
    </div>
  </div>
</template>

<script>

import videojs from 'video.js'
import 'videojs-contrib-hls'
import { fetchArticle } from '@/api/article'
import flvPlayer from 'flv.js'

const stream = {}
export default {
  name: 'Play',
  data() {
    return {
      flvPlayer: null,
      stream: Object.assign({}, stream)
    }
  },
  created() {
    const id = this.$route.params && this.$route.params.id
    this.fetchData(id)
  },
  mounted() {

  },
  methods: {
    fetchData(id) {
      fetchArticle(id).then(response => {
        this.stream = response.data
        var videoElement = document.getElementById('video-source')
        videoElement.setAttribute('src', this.stream.HlsUrl)
        setTimeout(() => {
          this.getVideo()
        }, 3 * 1000)
      }).catch(err => {
        console.log(err)
      })
    },
    getVideo() {
      videojs(
        'my-video',
        {
          bigPlayButton: false,
          textTrackDisplay: false,
          posterImage: true,
          errorDisplay: false,
          controlBar: true
        },
        function() {
          this.play()
        }
      )
    }
  }
}
</script>
