textHelpers       = require 'activity/util/textHelpers'
isWithinCodeBlock = require 'app/util/isWithinCodeBlock'
EmojiDropbox      = require 'activity/components/emojidropbox'

module.exports = EmojiToken =

  extractQuery: (value, position) ->

    return  if not value or isWithinCodeBlock value, position

    currentWord = textHelpers.getWordByPosition value, position
    return  unless currentWord

    matchResult = currentWord.match /^\:(.+)/
    return matchResult[1]  if matchResult


  getConfig: ->

    return {
      component            : EmojiDropbox
      getters              :
        items              : 'dropboxEmojis'
        selectedIndex      : 'emojisSelectedIndex'
        selectedItem       : 'emojisSelectedItem'
        formattedItem      : 'emojisFormattedItem'
      horizontalNavigation : yes
    }

