<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Close, Microphone, Mute, PhoneFilled, VideoCamera, VideoPause } from '@element-plus/icons-vue'
import { api } from '../api'
import { useChatStore } from '../store/chat'
import { useUserStore } from '../store/user'

const chat = useChatStore()
const user = useUserStore()

const visible = ref(false)
const phase = ref('idle')
const peerUid = ref('')
const peerName = ref('')
const callId = ref('')
const pendingOffer = ref('')
const localVideo = ref(null)
const remoteVideo = ref(null)
const localStream = ref(null)
const remoteStream = ref(null)
const micEnabled = ref(true)
const cameraEnabled = ref(true)

let pc = null
let pendingCandidates = []

const rtcConfig = {
  iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
}

const title = computed(() => {
  if (phase.value === 'incoming') return 'Incoming video call'
  if (phase.value === 'calling') return 'Calling'
  if (phase.value === 'connected') return 'Video call'
  return 'Connecting'
})

const statusText = computed(() => {
  const name = peerName.value || peerUid.value
  if (phase.value === 'incoming') return `${name} is inviting you to a video call`
  if (phase.value === 'calling') return `Waiting for ${name} to answer`
  if (phase.value === 'connected') return name
  return 'Establishing connection'
})

function makeCallId() {
  if (crypto && crypto.randomUUID) return crypto.randomUUID()
  return `${Date.now()}_${Math.random().toString(16).slice(2)}`
}

function friendName(uid) {
  const f = chat.friends.find((x) => x.uid === uid)
  return f ? f.name : uid
}

async function attachStreams() {
  await nextTick()
  if (localVideo.value) localVideo.value.srcObject = localStream.value
  if (remoteVideo.value) remoteVideo.value.srcObject = remoteStream.value
}

async function openMedia() {
  if (localStream.value) return
  if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
    throw new Error('Camera or microphone is not available')
  }
  localStream.value = await navigator.mediaDevices.getUserMedia({ video: true, audio: true })
  micEnabled.value = true
  cameraEnabled.value = true
  await attachStreams()
}

function createPeer() {
  if (pc) return pc
  pc = new RTCPeerConnection(rtcConfig)
  remoteStream.value = new MediaStream()
  pc.ontrack = (event) => {
    for (const track of event.streams[0]?.getTracks() || [event.track]) {
      if (!remoteStream.value.getTracks().find((t) => t.id === track.id)) {
        remoteStream.value.addTrack(track)
      }
    }
    attachStreams()
  }
  pc.onicecandidate = (event) => {
    if (event.candidate) {
      sendSignal('candidate', '', event.candidate.toJSON ? event.candidate.toJSON() : event.candidate)
    }
  }
  pc.onconnectionstatechange = () => {
    if (!pc) return
    if (pc.connectionState === 'connected') phase.value = 'connected'
    if (['failed', 'disconnected', 'closed'].includes(pc.connectionState)) {
      if (phase.value !== 'idle') closeCall(false)
    }
  }
  for (const track of localStream.value?.getTracks() || []) {
    pc.addTrack(track, localStream.value)
  }
  return pc
}

async function flushCandidates() {
  if (!pc || !pc.remoteDescription) return
  const list = pendingCandidates
  pendingCandidates = []
  for (const candidate of list) {
    try {
      await pc.addIceCandidate(candidate)
    } catch {
      /* ignore stale ICE */
    }
  }
}

function sendSignal(signalType, sdp = '', candidate = null) {
  if (!peerUid.value) return
  const candidateJSON = candidate ? JSON.stringify(candidate) : ''
  api.sendVideoSignal(peerUid.value, signalType, sdp, candidateJSON, callId.value).catch(() => {})
}

async function startOutgoing(req) {
  if (phase.value !== 'idle') return
  peerUid.value = req.peerUid
  peerName.value = req.peerName || friendName(req.peerUid)
  callId.value = makeCallId()
  visible.value = true
  phase.value = 'calling'
  try {
    await openMedia()
    const conn = createPeer()
    const offer = await conn.createOffer()
    await conn.setLocalDescription(offer)
    sendSignal('offer', offer.sdp || '')
  } catch (e) {
    ElMessage.error(String(e?.message || e))
    closeCall(false)
  }
}

function receiveOffer(signal) {
  if (phase.value !== 'idle') {
    api.sendVideoSignal(signal.from_uid, 'reject', '', '', signal.call_id || '').catch(() => {})
    return
  }
  peerUid.value = signal.from_uid
  peerName.value = friendName(signal.from_uid)
  callId.value = signal.call_id || makeCallId()
  pendingOffer.value = signal.sdp || ''
  visible.value = true
  phase.value = 'incoming'
}

async function acceptCall() {
  if (!pendingOffer.value) return
  phase.value = 'connecting'
  try {
    await openMedia()
    const conn = createPeer()
    await conn.setRemoteDescription({ type: 'offer', sdp: pendingOffer.value })
    await flushCandidates()
    const answer = await conn.createAnswer()
    await conn.setLocalDescription(answer)
    sendSignal('answer', answer.sdp || '')
  } catch (e) {
    ElMessage.error(String(e?.message || e))
    sendSignal('reject')
    closeCall(false)
  }
}

async function receiveAnswer(signal) {
  if (!pc || signal.call_id !== callId.value) return
  try {
    await pc.setRemoteDescription({ type: 'answer', sdp: signal.sdp || '' })
    await flushCandidates()
    phase.value = 'connecting'
  } catch {
    closeCall(true)
  }
}

async function receiveCandidate(signal) {
  if (!signal.candidate || signal.call_id !== callId.value) return
  const candidate = new RTCIceCandidate(signal.candidate)
  if (!pc || !pc.remoteDescription) {
    pendingCandidates.push(candidate)
    return
  }
  try {
    await pc.addIceCandidate(candidate)
  } catch {
    /* ignore stale ICE */
  }
}

function handleSignal(signal) {
  if (!signal || signal.from_uid === user.uid) return
  if (signal.signal_type === 'offer') receiveOffer(signal)
  else if (signal.signal_type === 'answer') receiveAnswer(signal)
  else if (signal.signal_type === 'candidate') receiveCandidate(signal)
  else if (['end', 'cancel', 'reject'].includes(signal.signal_type)) {
    if (!callId.value || signal.call_id === callId.value) closeCall(false)
  }
}

function toggleMic() {
  micEnabled.value = !micEnabled.value
  for (const track of localStream.value?.getAudioTracks() || []) track.enabled = micEnabled.value
}

function toggleCamera() {
  cameraEnabled.value = !cameraEnabled.value
  for (const track of localStream.value?.getVideoTracks() || []) track.enabled = cameraEnabled.value
}

function rejectCall() {
  sendSignal('reject')
  closeCall(false)
}

function closeCall(notifyPeer = true) {
  if (notifyPeer && peerUid.value && phase.value !== 'idle') sendSignal('end')
  if (pc) {
    pc.ontrack = null
    pc.onicecandidate = null
    pc.onconnectionstatechange = null
    pc.close()
    pc = null
  }
  for (const track of localStream.value?.getTracks() || []) track.stop()
  for (const track of remoteStream.value?.getTracks() || []) track.stop()
  localStream.value = null
  remoteStream.value = null
  pendingCandidates = []
  pendingOffer.value = ''
  visible.value = false
  phase.value = 'idle'
  peerUid.value = ''
  peerName.value = ''
  callId.value = ''
}

watch(
  () => chat.videoCallRequestSeq,
  () => {
    if (chat.videoCallRequest) startOutgoing(chat.videoCallRequest)
  }
)

watch(
  () => chat.videoSignalSeq,
  () => handleSignal(chat.lastVideoSignal)
)

onBeforeUnmount(() => closeCall(true))
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="title"
    width="680px"
    align-center
    :close-on-click-modal="false"
    :show-close="false"
    class="video-dialog"
  >
    <div class="call-stage" :class="{ waiting: phase !== 'connected' }">
      <video ref="remoteVideo" class="remote-video" autoplay playsinline></video>
      <div v-if="phase !== 'connected'" class="call-status">{{ statusText }}</div>
      <video ref="localVideo" class="local-video" autoplay muted playsinline></video>
    </div>

    <div class="call-controls">
      <template v-if="phase === 'incoming'">
        <el-button type="danger" circle :icon="Close" title="Reject" @click="rejectCall" />
        <el-button type="success" circle :icon="PhoneFilled" title="Accept" @click="acceptCall" />
      </template>
      <template v-else>
        <el-button circle :icon="micEnabled ? Microphone : Mute" title="Microphone" @click="toggleMic" />
        <el-button circle :icon="cameraEnabled ? VideoCamera : VideoPause" title="Camera" @click="toggleCamera" />
        <el-button type="danger" circle :icon="Close" title="End" @click="closeCall(true)" />
      </template>
    </div>
  </el-dialog>
</template>

<style scoped>
.call-stage {
  position: relative;
  overflow: hidden;
  width: 100%;
  aspect-ratio: 16 / 9;
  background: #111318;
  border-radius: 8px;
}
.remote-video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  background: #111318;
}
.call-stage.waiting .remote-video {
  opacity: 0.18;
}
.local-video {
  position: absolute;
  right: 14px;
  bottom: 14px;
  width: 168px;
  aspect-ratio: 4 / 3;
  object-fit: cover;
  background: #252932;
  border: 2px solid rgba(255, 255, 255, 0.75);
  border-radius: 6px;
}
.call-status {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 88px;
  color: #fff;
  font-size: 18px;
  text-align: center;
}
.call-controls {
  display: flex;
  justify-content: center;
  gap: 16px;
  padding-top: 18px;
}

@media (max-width: 720px) {
  .local-video {
    width: 120px;
  }
  .call-status {
    padding: 0 28px;
    font-size: 15px;
  }
}
</style>