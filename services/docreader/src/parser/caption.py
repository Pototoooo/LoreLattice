import json
import logging
import os
from dataclasses import dataclass, field
from typing import List, Optional, Union

import requests

logger = logging.getLogger(__name__)


@dataclass
class ImageUrl:
    """Image URL data structure for caption requests."""

    url: Optional[str] = None
    detail: Optional[str] = None


@dataclass
class Content:
    """Content data structure that can contain text or image URL."""

    type: Optional[str] = None
    text: Optional[str] = None
    image_url: Optional[ImageUrl] = None


@dataclass
class SystemMessage:
    """System message for VLM model requests."""

    role: Optional[str] = None
    content: Optional[str] = None


@dataclass
class UserMessage:
    """User message for VLM model requests, can contain multiple content items."""

    role: Optional[str] = None
    content: List[Content] = field(default_factory=list)


@dataclass
class CompletionRequest:
    """Request structure for VLM model completion API."""

    model: str
    temperature: float
    top_p: float
    messages: List[Union[SystemMessage, UserMessage]]
    user: str


@dataclass
class Model:
    """Model identifier structure."""

    id: str


@dataclass
class ModelsResp:
    """Response structure for available models API."""

    data: List[Model] = field(default_factory=list)


@dataclass
class Message:
    """Message structure in API response."""

    role: Optional[str] = None
    content: Optional[str] = None
    tool_calls: Optional[str] = None


@dataclass
class Choice:
    """Choice structure in API response."""

    message: Optional[Message] = None


@dataclass
class Usage:
    """Token usage information in API response."""

    prompt_tokens: Optional[int] = 0
    total_tokens: Optional[int] = 0
    completion_tokens: Optional[int] = 0


@dataclass
class CaptionChatResp:
    """Response structure for caption chat API."""

    id: Optional[str] = None
    created: Optional[int] = None
    model: Optional[Model] = None
    object: Optional[str] = None
    choices: List[Choice] = field(default_factory=list)
    usage: Optional[Usage] = None

    @staticmethod
    def from_json(json_data: dict) -> "CaptionChatResp":
        """
        Parse API response JSON into a CaptionChatResp object.

        Args:
            json_data: The JSON response from the API

        Returns:
            A parsed CaptionChatResp object
        """
        logger.info("Parsing CaptionChatResp from JSON")
        # Manually parse nested fields with safe field extraction
        choices = []
        for choice in json_data.get("choices", []):
            message_data = choice.get("message", {})
            message = Message(
                role=message_data.get("role"),
                content=message_data.get("content"),
                tool_calls=message_data.get("tool_calls"),
            )
            choices.append(Choice(message=message))

        # Handle usage with safe field extraction
        usage_data = json_data.get("usage", {})
        usage = None
        if usage_data:
            usage = Usage(
                prompt_tokens=usage_data.get("prompt_tokens", 0),
                total_tokens=usage_data.get("total_tokens", 0),
                completion_tokens=usage_data.get("completion_tokens", 0),
            )

        logger.info(
            f"Parsed {len(choices)} choices and usage data: {usage is not None}"
        )
        return CaptionChatResp(
            id=json_data.get("id"),
            created=json_data.get("created"),
            model=json_data.get("model"),
            object=json_data.get("object"),
            choices=choices,
            usage=usage,
        )

    def choice_data(self) -> str:
        """
        Extract the content from the first choice in the response.

        Returns:
            The content string from the first choice, or empty string if no choices
        """
        if self.choices:
            logger.info("Retrieving content from first choice")
            return self.choices[0].message.content
        logger.warning("No choices available in response")
        return ""


class Caption:
    """
    Service for generating captions for images using a Vision Language Model.
    Uses an external API to process images and return textual descriptions.
    """

    def __init__(self):
        """Initialize the Caption service with configuration from environment variables."""
        logger.info("Initializing Caption service")
        self.prompt = """简单凝炼的描述图片的主要内容"""
        if os.getenv("VLM_MODEL_BASE_URL") == "" or os.getenv("VLM_MODEL_NAME") == "":
            logger.error("VLM_MODEL_BASE_URL or VLM_MODEL_NAME is not set")
            return
        self.completion_url = os.getenv("VLM_MODEL_BASE_URL") + "/v1/chat/completions"
        self.model = os.getenv("VLM_MODEL_NAME")
        logger.info(
            f"Service configured with model: {self.model}, endpoint: {self.completion_url}"
        )

    def _call_caption_api(self, image_url: str) -> Optional[CaptionChatResp]:
        """
        Call the Caption API to generate a description for the given image.

        Args:
            image_url: URL of the image to be captioned

        Returns:
            CaptionChatResp object if successful, None otherwise
        """
        logger.info(f"Calling Caption API for image captioning")
        logger.info(f"Processing image from URL: {image_url}")

        user_msg = UserMessage(
            role="user",
            content=[
                Content(type="text", text=self.prompt),
                Content(
                    type="image_url", image_url=ImageUrl(url=image_url, detail="auto")
                ),
            ],
        )

        gpt_req = CompletionRequest(
            model=self.model,
            temperature=0.3,
            top_p=0.8,
            messages=[user_msg],
            user="abc",
        )

        headers = {
            "Content-Type": "application/json",
            "Accept": "text/event-stream",
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
        }

        try:
            logger.info(f"Sending request to Caption API with model: {self.model}")
            response = requests.post(
                self.completion_url,
                data=json.dumps(gpt_req, default=lambda o: o.__dict__, indent=4),
                headers=headers,
                timeout=30,
            )
            if response.status_code != 200:
                logger.error(
                    f"Caption API returned non-200 status code: {response.status_code}"
                )
                response.raise_for_status()

            logger.info(
                f"Successfully received response from Caption API with status: {response.status_code}"
            )
            logger.info(f"Converting response to CaptionChatResp object")
            caption_resp = CaptionChatResp.from_json(response.json())

            if caption_resp.usage:
                logger.info(
                    f"API usage: prompt_tokens={caption_resp.usage.prompt_tokens}, "
                    f"completion_tokens={caption_resp.usage.completion_tokens}"
                )

            return caption_resp
        except requests.exceptions.Timeout:
            logger.error(f"Timeout while calling Caption API after 30 seconds")
            return None
        except requests.exceptions.RequestException as e:
            logger.error(f"Request error calling Caption API: {e}")
            return None
        except Exception as e:
            logger.error(f"Error calling Caption API: {str(e)}")
            logger.info(
                f"Request details: model={self.model}, URL={self.completion_url}"
            )
            return None

    def get_caption(self, image_url: str) -> str:
        """
        Get a caption for the provided image URL.

        Args:
            image_url: URL of the image to be captioned

        Returns:
            Caption text as string, or empty string if captioning failed
        """
        logger.info("Getting caption for image")
        if not image_url or self.completion_url is None:
            logger.error("Image URL is not set")
            return ""
        caption_resp = self._call_caption_api(image_url)
        if caption_resp:
            caption = caption_resp.choice_data()
            caption_length = len(caption)
            logger.info(f"Successfully generated caption of length {caption_length}")
            logger.info(
                f"Caption: {caption[:50]}..."
                if caption_length > 50
                else f"Caption: {caption}"
            )
            return caption
        logger.warning("Failed to get caption from Caption API")
        return ""
